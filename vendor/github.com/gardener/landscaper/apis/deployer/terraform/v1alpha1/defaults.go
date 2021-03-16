// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_Configuration sets the defaults for the terraform deployer configuration.
func SetDefaults_Configuration(obj *Configuration) {
	DefaultConfiguration(obj, "")
}

// DefaultConfiguration defaults the configuration with the current version of the deployer.
func DefaultConfiguration(obj *Configuration, version string) {
	if len(obj.Namespace) == 0 {
		obj.Namespace = metav1.NamespaceDefault
	}
	if len(obj.Terraformer.TerraformContainer.Image) == 0 {
		// TODO: add a component reference to the terraformer component.
		obj.Terraformer.TerraformContainer.Image = "eu.gcr.io/gardener-project/gardener/terraformer:v2.0.0"
	}
	if len(obj.Terraformer.LogLevel) == 0 {
		obj.Terraformer.LogLevel = "info"
	}

	if len(version) != 0 {
		if len(obj.Terraformer.TerraformContainer.Image) == 0 {
			obj.Terraformer.TerraformContainer.Image = "eu.gcr.io/gardener-project/landscaper/terraform-deployer-init:" + version
		}
	}
}
