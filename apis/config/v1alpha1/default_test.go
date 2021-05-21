// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	"github.com/gardener/landscaper/apis/config/v1alpha1"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "v1alpha1 Test Suite")
}

var _ = Describe("Defaults", func() {

	It("should default crd management", func() {
		cfg := &v1alpha1.CrdManagementConfiguration{}
		v1alpha1.SetDefaults_CrdManagementConfiguration(cfg)
		Expect(cfg.DeployCustomResourceDefinitions).To(gstruct.PointTo(Equal(true)))
		Expect(cfg.ForceUpdate).To(gstruct.PointTo(Equal(true)))
	})

})
