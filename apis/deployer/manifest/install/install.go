// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/landscaper/apis/deployer/manifest"
	"github.com/gardener/landscaper/apis/deployer/manifest/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
)

var (
	schemeBuilder = runtime.NewSchemeBuilder(
		v1alpha2.AddToScheme,
		v1alpha1.AddToScheme,
		manifest.AddToScheme,
		setVersionPriority,
	)

	AddToScheme = schemeBuilder.AddToScheme
)

func setVersionPriority(scheme *runtime.Scheme) error {
	return scheme.SetVersionPriority(v1alpha2.SchemeGroupVersion)
}

// Install installs all APIs in the scheme.
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(AddToScheme(scheme))
}
