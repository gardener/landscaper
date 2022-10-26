// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"
	"time"

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

	Context("BlueprintStore", func() {

		It("should default index method", func() {
			cfg := &v1alpha1.BlueprintStore{}
			v1alpha1.SetDefaults_BlueprintStore(cfg)
			Expect(cfg.IndexMethod).To(Equal(v1alpha1.BlueprintDigestIndex))
		})

	})

	Context("CommonControllerConfig", func() {

		checkCommonConfig := func(cfg *v1alpha1.CommonControllerConfig) {
			Expect(cfg.Workers).To(Equal(1))
			Expect(cfg.CacheSyncTimeout).ToNot(BeNil())
			Expect(cfg.CacheSyncTimeout.Duration).To(Equal(2 * time.Minute))
		}

		It("should default the common controller config", func() {
			cfg := &v1alpha1.CommonControllerConfig{}
			v1alpha1.SetDefaults_CommonControllerConfig(cfg)
			checkCommonConfig(cfg)
		})

		It("should default the controllers commonconfig", func() {
			cfg := &v1alpha1.LandscaperConfiguration{}
			v1alpha1.SetDefaults_LandscaperConfiguration(cfg)
			checkCommonConfig(&cfg.Controllers.Installations.CommonControllerConfig)
			checkCommonConfig(&cfg.Controllers.Executions.CommonControllerConfig)
			checkCommonConfig(&cfg.Controllers.DeployItems.CommonControllerConfig)
			checkCommonConfig(&cfg.Controllers.Contexts.CommonControllerConfig)
		})
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
