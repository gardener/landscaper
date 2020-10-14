// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubernetes utils Test Suite")
}

var _ = Describe("Kubernetes", func() {

	Context("OwnerOfGVK", func() {

		It("should validate that a secret is owned by another secret", func() {
			secret := &corev1.Secret{}
			secret.Name = "child"
			secret.Namespace = "default"
			owner := &corev1.Secret{}
			owner.Name = "owner"
			owner.Namespace = "default"
			err := controllerutil.SetOwnerReference(owner, secret, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			gvk, err := apiutil.GVKForObject(owner, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			ownerName, isOwned := kubernetes.OwnerOfGVK(secret.GetOwnerReferences(), gvk)
			Expect(isOwned).To(BeTrue())
			Expect(ownerName).To(Equal("owner"))
		})

		It("should validate that a secret is not owner by a configmap", func() {

			secret := &corev1.Secret{}
			secret.Name = "child"
			secret.Namespace = "default"
			owner := &corev1.Secret{}
			owner.Name = "owner"
			owner.Namespace = "default"
			err := controllerutil.SetOwnerReference(owner, secret, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			gvk, err := apiutil.GVKForObject(&corev1.ConfigMap{}, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			_, isOwned := kubernetes.OwnerOfGVK(secret.GetOwnerReferences(), gvk)
			Expect(isOwned).To(BeFalse())
		})

	})

})
