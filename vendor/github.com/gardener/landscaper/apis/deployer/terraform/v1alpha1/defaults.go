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
	if len(obj.Terraformer.Namespace) == 0 {
		obj.Terraformer.Namespace = metav1.NamespaceDefault
	}
	if len(obj.Terraformer.Image) == 0 {
		// TODO: add a component reference to the terraformer component.
		obj.Terraformer.Image = "eu.gcr.io/gardener-project/gardener/terraformer:v2.0.0"
	}
	if len(obj.Terraformer.LogLevel) == 0 {
		obj.Terraformer.LogLevel = "info"
	}
}
