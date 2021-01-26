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
	if obj.DefaultOCI != nil {
		if obj.Registry.OCI == nil {
			obj.Registry.OCI = obj.DefaultOCI
		}
	}
}
