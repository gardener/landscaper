// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/landscaper/utils/kubernetes"
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
