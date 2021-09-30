// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
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

	It("should default the repository context in the context controller", func() {
		repoCtx, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com", ""))
		cfg := &v1alpha1.LandscaperConfiguration{}
		cfg.RepositoryContext = &repoCtx
		v1alpha1.SetDefaults_LandscaperConfiguration(cfg)
		Expect(cfg.Controllers.Contexts.Config.Default.RepositoryContext).ToNot(BeNil())
		Expect(cfg.Controllers.Contexts.Config.Default.RepositoryContext.Raw).To(MatchJSON(repoCtx.Raw))
	})

})
