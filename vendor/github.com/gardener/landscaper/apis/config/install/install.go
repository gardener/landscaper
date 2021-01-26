// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
)

var (
	schemeBuilder = runtime.NewSchemeBuilder(
		v1alpha1.AddToScheme,
		config.AddToScheme,
		setVersionPriority,
	)

	AddToScheme = schemeBuilder.AddToScheme
)

func setVersionPriority(scheme *runtime.Scheme) error {
	return scheme.SetVersionPriority(v1alpha1.SchemeGroupVersion)
}

// Install installs all APIs in the scheme.
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(AddToScheme(scheme))
}
