// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
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

// SetDefaults_Configuration sets the defaults for the container deployer configuration.
func SetDefaults_Configuration(obj *Configuration) {
	if len(obj.Namespace) == 0 {
		obj.Namespace = metav1.NamespaceDefault
	}
	if len(obj.DefaultImage.Image) == 0 {
		obj.DefaultImage.Image = "ubuntu:18.04"
	}
	SetDefaults_GarbageCollection(&obj.GarbageCollection)
}

// SetDefaults_GarbageCollection sets the defaults for the container deployer configuration.
func SetDefaults_GarbageCollection(obj *GarbageCollection) {
	if obj.Worker <= 0 {
		obj.Worker = 5
	}
	if obj.RequeueTimeSeconds <= 0 {
		obj.RequeueTimeSeconds = 60 * 60
	}
}
