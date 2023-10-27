// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	lsconfigv1alpha1 "github.com/gardener/landscaper/apis/config/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_ProviderConfiguration sets the defaults for the manifest deployer provider configuration.
func SetDefaults_ProviderConfiguration(obj *ProviderConfiguration) {
	if len(obj.UpdateStrategy) == 0 {
		obj.UpdateStrategy = UpdateStrategyUpdate
	}
}

// SetDefaults_Configuration sets the defaults for the manifest deployer controller configuration.
func SetDefaults_Configuration(obj *Configuration) {
	lsconfigv1alpha1.SetDefaults_CommonControllerConfig(&obj.Controller.CommonControllerConfig)
}
