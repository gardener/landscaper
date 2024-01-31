// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_LandscaperConfiguration sets the defaults for the landscaper configuration.
func SetDefaults_LandscaperConfiguration(obj *LandscaperConfiguration) {
	if obj.Registry.OCI == nil {
		obj.Registry.OCI = &OCIConfiguration{}
	}
	if obj.Registry.OCI.Cache == nil {
		obj.Registry.OCI.Cache = &OCICacheConfiguration{
			UseInMemoryOverlay: false,
		}
	}

	SetDefaults_CommonControllerConfig(&obj.Controllers.Installations.CommonControllerConfig)
	SetDefaults_CommonControllerConfig(&obj.Controllers.Executions.CommonControllerConfig)
	SetDefaults_CommonControllerConfig(&obj.Controllers.DeployItems.CommonControllerConfig)
	SetDefaults_CommonControllerConfig(&obj.Controllers.Contexts.CommonControllerConfig)

	if len(obj.DeployerManagement.Namespace) == 0 {
		obj.DeployerManagement.Namespace = "ls-system"
	}
	if obj.DeployItemTimeouts == nil {
		obj.DeployItemTimeouts = &DeployItemTimeouts{}
	}
	if obj.DeployItemTimeouts.Pickup == nil {
		obj.DeployItemTimeouts.Pickup = &v1alpha1.Duration{Duration: 5 * time.Minute}
	}
	if obj.DeployItemTimeouts.Abort == nil {
		obj.DeployItemTimeouts.Abort = &v1alpha1.Duration{Duration: 5 * time.Minute}
	}

	SetDefaults_BlueprintStore(&obj.BlueprintStore)
	SetDefaults_CrdManagementConfiguration(&obj.CrdManagement)

	if obj.RepositoryContext != nil && obj.Controllers.Contexts.Config.Default.RepositoryContext == nil {
		// migrate the repository context to the new structure.
		// The old location is ignored if a repository context is defined in the new location.
		obj.Controllers.Contexts.Config.Default.RepositoryContext = obj.RepositoryContext
	}

	if obj.DeployerManagement.Disable {
		obj.DeployerManagement.Agent.Disable = true
	}
	if len(obj.DeployerManagement.Agent.Name) == 0 {
		obj.DeployerManagement.Agent.Name = "default"
	}
	if len(obj.DeployerManagement.Agent.Namespace) == 0 {
		obj.DeployerManagement.Agent.Namespace = obj.DeployerManagement.Namespace
	}
	if len(obj.DeployerManagement.Agent.LandscaperNamespace) == 0 {
		obj.DeployerManagement.Agent.LandscaperNamespace = obj.DeployerManagement.Namespace
	}
	if obj.DeployerManagement.Agent.OCI == nil {
		obj.DeployerManagement.Agent.OCI = obj.Registry.OCI
	}

	if obj.DeployerManagement.DeployerRepositoryContext == nil {
		defaultCtx, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper", ""))
		obj.DeployerManagement.DeployerRepositoryContext = &defaultCtx
	}
}

// SetDefaults_CrdManagementConfiguration sets the defaults for the crd management configuration.
func SetDefaults_CrdManagementConfiguration(obj *CrdManagementConfiguration) {
	if obj.DeployCustomResourceDefinitions == nil {
		obj.DeployCustomResourceDefinitions = pointer.Bool(true)
	}
	if obj.ForceUpdate == nil {
		obj.ForceUpdate = pointer.Bool(true)
	}
}

// SetDefaults_CommonControllerConfig sets the defaults for the CommonControllerConfig.
func SetDefaults_CommonControllerConfig(obj *CommonControllerConfig) {
	if obj.Workers == 0 {
		obj.Workers = 1
	}
	if obj.CacheSyncTimeout == nil {
		obj.CacheSyncTimeout = &metav1.Duration{
			Duration: 2 * time.Minute,
		}
	}
}

// SetDefaults_AgentConfiguration sets the defaults for the landscaper configuration.
func SetDefaults_AgentConfiguration(obj *AgentConfiguration) {
	if len(obj.Namespace) == 0 {
		obj.Namespace = "ls-system"
	}
	if len(obj.LandscaperNamespace) == 0 {
		obj.LandscaperNamespace = "ls-system"
	}
}

// SetDefaults_BlueprintStore sets the defaults for the landscaper blueprint store configuration.
func SetDefaults_BlueprintStore(obj *BlueprintStore) {
	// GCHighThreshold defines the default percent of disk usage which triggers files garbage collection.
	const GCHighThreshold float64 = 0.85

	// GCLowThreshold defines the default percent of disk usage to which files garbage collection attempts to free.
	const GCLowThreshold float64 = 0.80

	// ResetInterval defines the default interval when the hit reset should run.
	ResetInterval := metav1.Duration{Duration: 1 * time.Hour}

	// PreservedHitsProportion defines the default percent of hits that should be preserved.
	const PreservedHitsProportion = 0.5

	if len(obj.IndexMethod) == 0 {
		obj.IndexMethod = BlueprintDigestIndex
	}

	if obj.Size == "0" {
		// no garbage collection configured ignore all other values
		return
	}

	if len(obj.Size) == 0 {
		obj.Size = "250Mi"
	}

	if obj.GCHighThreshold == 0 {
		obj.GCHighThreshold = GCHighThreshold
	}
	if obj.GCLowThreshold == 0 {
		obj.GCLowThreshold = GCLowThreshold
	}

	if obj.ResetInterval == nil {
		obj.ResetInterval = &ResetInterval
	}

	if obj.PreservedHitsProportion == 0 {
		obj.PreservedHitsProportion = PreservedHitsProportion
	}
}
