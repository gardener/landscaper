// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

var _ = g.Describe("helper", func() {

	g.Context("IsRoot", func() {

		g.It("should validate that a installation with a non Installation type owner is a root installation", func() {
			inst := &lsv1alpha1.Installation{}
			inst.Name = "inst"
			inst.Namespace = "default"
			inst.Labels = map[string]string{lsv1alpha1.EncompassedByLabel: "owner"}

			owner := &corev1.Secret{}
			owner.Name = "owner"
			owner.Namespace = "default"
			err := controllerutil.SetOwnerReference(owner, inst, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			isRoot := IsRootInstallation(inst)
			Expect(isRoot).To(BeTrue())
		})

		g.It("should validate that a installation with a installation owner is not a root installation", func() {
			inst := &lsv1alpha1.Installation{}
			inst.Name = "inst"
			inst.Namespace = "default"
			inst.Labels = map[string]string{lsv1alpha1.EncompassedByLabel: "owner"}

			owner := &lsv1alpha1.Installation{}
			owner.Name = "owner"
			owner.Namespace = "default"
			err := controllerutil.SetOwnerReference(owner, inst, kubernetes.LandscaperScheme)
			Expect(err).ToNot(HaveOccurred())

			isRoot := IsRootInstallation(inst)
			Expect(isRoot).To(BeFalse())
		})
	})

})
