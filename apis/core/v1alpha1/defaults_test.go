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
	})

})
