// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_Configuration sets the defaults for the terraform deployer configuration.
func SetDefaults_ProviderConfiguration(obj *ProviderConfiguration) {
	if len(obj.Namespace) == 0 {
		obj.Namespace = metav1.NamespaceDefault
	}
	if len(obj.TerraformerImage) == 0 {
		obj.TerraformerImage = "eu.gcr.io/gardener-project/gardener/terraformer:v1.5.0"
	}
}
