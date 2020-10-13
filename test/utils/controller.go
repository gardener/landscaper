// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// ShouldReconcile reconciles the given reconciler with the given request
// and expects that no error occurred
func ShouldReconcile(reconciler reconcile.Reconciler, req reconcile.Request, optionalDescription ...interface{}) {
	_, err := reconciler.Reconcile(req)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred(), optionalDescription...)
}

// ShouldNotReconcile reconciles the given reconciler with the given request
// and expects that an error occurred
func ShouldNotReconcile(reconciler reconcile.Reconciler, req reconcile.Request, optionalDescription ...interface{}) error {
	_, err := reconciler.Reconcile(req)
	gomega.ExpectWithOffset(1, err).To(gomega.HaveOccurred(), optionalDescription...)
	return err
}

// Request creates a new reconcile.Request
func Request(name, namespace string) reconcile.Request {
	req := reconcile.Request{}
	req.Name = name
	req.Namespace = namespace
	return req
}

// DeleteInstallation deletes a component by reconciling it with the expected reconcile loops
func DeleteInstallation(ctx context.Context, client client.Client, execActuator, instActuator, mockActuator reconcile.Reconciler, instReq reconcile.Request) {
	inst := &lsv1alpha1.Installation{}
	gomega.Expect(client.Get(ctx, instReq.NamespacedName, inst)).ToNot(gomega.HaveOccurred())
	gomega.Expect(client.Delete(ctx, inst)).ToNot(gomega.HaveOccurred())

	execReq := Request(inst.Status.ExecutionReference.Name, inst.Status.ExecutionReference.Namespace)
	exec := &lsv1alpha1.Execution{}
	gomega.Expect(client.Get(ctx, execReq.NamespacedName, exec)).ToNot(gomega.HaveOccurred())

	// the installation controller should propagate the deletion to its subcharts
	err := ShouldNotReconcile(instActuator, instReq)
	gomega.Expect(err.Error()).To(gomega.ContainSubstring("waiting for deletion"))

	gomega.Expect(client.Get(ctx, execReq.NamespacedName, exec)).ToNot(gomega.HaveOccurred())
	gomega.Expect(exec.DeletionTimestamp.IsZero()).To(gomega.BeFalse(), "deletion timestamp should be set")

	// the execution controller should propagate the deletion to its deploy item
	ShouldReconcile(execActuator, execReq)

	diList := &lsv1alpha1.DeployItemList{}
	gomega.Expect(client.List(ctx, diList)).ToNot(gomega.HaveOccurred())

	for i := len(exec.Status.DeployItemReferences) - 1; i >= 0; i-- {
		diRef := exec.Status.DeployItemReferences[i]
		diReq := Request(diRef.Reference.Name, diRef.Reference.Namespace)
		di := &lsv1alpha1.DeployItem{}
		gomega.Expect(client.Get(ctx, diRef.Reference.NamespacedName(), di)).ToNot(gomega.HaveOccurred())
		gomega.Expect(di.DeletionTimestamp.IsZero()).To(gomega.BeFalse(), "deletion timestamp should be set")

		ShouldReconcile(mockActuator, diReq)
		err = client.Get(ctx, diReq.NamespacedName, di)
		gomega.Expect(err).To(gomega.HaveOccurred())
		gomega.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue(), "deploy item should be deleted")
	}

	// execution controller should remove the finalizer
	ShouldReconcile(execActuator, execReq)
	err = client.Get(ctx, execReq.NamespacedName, exec)
	gomega.Expect(err).To(gomega.HaveOccurred())
	gomega.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue(), "execution should be deleted")

	// installation controller should remove its own finalizer
	ShouldReconcile(instActuator, instReq)
	err = client.Get(ctx, instReq.NamespacedName, inst)
	gomega.Expect(err).To(gomega.HaveOccurred())
	gomega.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue(), "installation should be deleted")
}
