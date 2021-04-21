// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
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
	if obj.DeployItemTimeouts == nil {
		obj.DeployItemTimeouts = &DeployItemTimeouts{}
	}
	if len(obj.DeployItemTimeouts.Pickup) == 0 {
		obj.DeployItemTimeouts.Pickup = "5m"
	}
	if len(obj.DeployItemTimeouts.Abort) == 0 {
		obj.DeployItemTimeouts.Abort = "5m"
	}
	if len(obj.DeployItemTimeouts.ProgressingDefault) == 0 {
		obj.DeployItemTimeouts.ProgressingDefault = "10m"
	}

}
