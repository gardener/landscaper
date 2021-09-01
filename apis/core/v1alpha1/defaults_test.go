// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

var _ = Describe("Defaults", func() {

	Context("Installation", func() {
		It("should default the context", func() {
			inst := &lsv1alpha1.Installation{}
			lsv1alpha1.SetDefaults_Installation(inst)
			Expect(inst.Spec.Context).To(Equal(lsv1alpha1.DefaultContextName))
		})

		It("should default the namespace of data imports from secrets", func() {
			inst := &lsv1alpha1.Installation{}
			inst.Namespace = "test"
			inst.Spec.Imports.Data = []lsv1alpha1.DataImport{
				{
					SecretRef: &lsv1alpha1.SecretReference{
						ObjectReference: lsv1alpha1.ObjectReference{
							Name: "abc",
						},
					},
				},
				{
					ConfigMapRef: &lsv1alpha1.ConfigMapReference{
						ObjectReference: lsv1alpha1.ObjectReference{
							Name: "abc",
						},
					},
				},
			}
			lsv1alpha1.SetDefaults_Installation(inst)
			Expect(inst.Spec.Imports.Data[0].SecretRef.Namespace).To(Equal("test"))
		})

		It("should default the namespace of data imports from configmaps", func() {
			inst := &lsv1alpha1.Installation{}
			inst.Namespace = "test"
			inst.Spec.Imports.Data = []lsv1alpha1.DataImport{
				{
					ConfigMapRef: &lsv1alpha1.ConfigMapReference{
						ObjectReference: lsv1alpha1.ObjectReference{
							Name: "abc",
						},
					},
				},
			}
			lsv1alpha1.SetDefaults_Installation(inst)
			Expect(inst.Spec.Imports.Data[0].SecretRef.Namespace).To(Equal("test"))
		})
	})

})
