// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_ProviderConfiguration sets the defaults for the manifest deployer provider configuration.
func SetDefaults_ProviderConfiguration(obj *ProviderConfiguration) {
	if len(obj.UpdateStrategy) == 0 {
		obj.UpdateStrategy = UpdateStrategyUpdate
	}
	if obj.HealthChecks.Timeout == nil {
		obj.HealthChecks.Timeout = &lsv1alpha1.Duration{Duration: 180 * time.Second}
	}
	if obj.DeleteTimeout == nil {
		obj.DeleteTimeout = &lsv1alpha1.Duration{Duration: 180 * time.Second}
	}
	for i := range obj.Manifests {
		if len(obj.Manifests[i].Policy) == 0 {
			obj.Manifests[i].Policy = ManagePolicy
		}
	}
}
