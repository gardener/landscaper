// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"time"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ShouldReconcile reconciles the given reconciler with the given request
// and expects that no error occurred
func ShouldReconcile(ctx context.Context, reconciler reconcile.Reconciler, req reconcile.Request, optionalDescription ...interface{}) {
	_, err := reconciler.Reconcile(ctx, req)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred(), optionalDescription...)
}

// ShouldNotReconcile reconciles the given reconciler with the given request
// and expects that an error occurred
func ShouldNotReconcile(ctx context.Context, reconciler reconcile.Reconciler, req reconcile.Request, optionalDescription ...interface{}) error {
	_, err := reconciler.Reconcile(ctx, req)
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
	err := ShouldNotReconcile(ctx, instActuator, instReq)
	gomega.Expect(err.Error()).To(gomega.ContainSubstring("waiting for deletion"))

	gomega.Expect(client.Get(ctx, execReq.NamespacedName, exec)).ToNot(gomega.HaveOccurred())
	gomega.Expect(exec.DeletionTimestamp.IsZero()).To(gomega.BeFalse(), "deletion timestamp should be set")

	// the execution controller should propagate the deletion to its deploy item
	ShouldReconcile(ctx, execActuator, execReq)

	diList := &lsv1alpha1.DeployItemList{}
	gomega.Expect(client.List(ctx, diList)).ToNot(gomega.HaveOccurred())

	for i := len(exec.Status.DeployItemReferences) - 1; i >= 0; i-- {
		diRef := exec.Status.DeployItemReferences[i]
		diReq := Request(diRef.Reference.Name, diRef.Reference.Namespace)
		di := &lsv1alpha1.DeployItem{}
		gomega.Expect(client.Get(ctx, diRef.Reference.NamespacedName(), di)).ToNot(gomega.HaveOccurred())
		gomega.Expect(di.DeletionTimestamp.IsZero()).To(gomega.BeFalse(), "deletion timestamp should be set")

		ShouldReconcile(ctx, mockActuator, diReq)
		err = client.Get(ctx, diReq.NamespacedName, di)
		gomega.Expect(err).To(gomega.HaveOccurred())
		gomega.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue(), "deploy item should be deleted")
	}

	// execution controller should remove the finalizer
	ShouldReconcile(ctx, execActuator, execReq)
	err = client.Get(ctx, execReq.NamespacedName, exec)
	gomega.Expect(err).To(gomega.HaveOccurred())
	gomega.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue(), "execution should be deleted")

	// installation controller should remove its own finalizer
	ShouldReconcile(ctx, instActuator, instReq)
	err = client.Get(ctx, instReq.NamespacedName, inst)
	gomega.Expect(err).To(gomega.HaveOccurred())
	gomega.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue(), "installation should be deleted")
}

// MimicKCMServiceAccountArgs describes the configuration for mimicing the service account behavior of the
// MimicKCMServiceAccount func.
type MimicKCMServiceAccountArgs struct {
	Name      string
	Namespace string
	Token     string

	Timeout time.Duration
}

// MimicKCMServiceAccount mimics the kube controller manager behavior for service accounts.
// The apiserver is watched for service accounts and a account secret is created.
func MimicKCMServiceAccount(ctx context.Context, client client.Client, args MimicKCMServiceAccountArgs) {
	if len(args.Token) == 0 {
		args.Token = "my-test-token"
	}
	if args.Timeout == 0 {
		args.Timeout = 20 * time.Second
	}
	go func() {
		_ = wait.PollImmediate(2*time.Second, args.Timeout, func() (done bool, err error) {
			// mimics the kube-controller-manager that creates a secret for the service account
			sa := &corev1.ServiceAccount{}
			if err := client.Get(ctx, kutil.ObjectKey(args.Name, args.Namespace), sa); err != nil {
				return false, nil
			}

			secret := &corev1.Secret{}
			secret.GenerateName = "my-sa-secret-"
			secret.Namespace = args.Namespace
			secret.Type = corev1.SecretTypeServiceAccountToken
			secret.Annotations = map[string]string{
				corev1.ServiceAccountNameKey: sa.Name,
			}
			secret.Data = map[string][]byte{
				corev1.ServiceAccountTokenKey: []byte(args.Token),
			}
			if err := client.Create(ctx, secret); err != nil {
				if !apierrors.IsAlreadyExists(err) {
					return false, nil
				}
			}

			sa.Secrets = []corev1.ObjectReference{
				{
					Kind:       "Secret",
					Namespace:  secret.Namespace,
					Name:       secret.Name,
					UID:        secret.UID,
					APIVersion: "",
				},
			}
			if err := client.Update(ctx, sa); err != nil {
				return false, nil
			}
			return true, nil
		})
	}()
}
