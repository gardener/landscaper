// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/landscaper/pkg/version"
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
	if len(obj.InitContainer.Image) == 0 {
		obj.InitContainer.Image = fmt.Sprintf("eu.gcr.io/gardener-project/landscaper/container-deployer-init:%s", version.Get().GitVersion)
	}
	if len(obj.WaitContainer.Image) == 0 {
		obj.WaitContainer.Image = fmt.Sprintf("eu.gcr.io/gardener-project/landscaper/container-deployer-wait:%s", version.Get().GitVersion)
	}
}
