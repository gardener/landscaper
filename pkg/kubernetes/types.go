// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	corescheme "k8s.io/client-go/kubernetes/scheme"

	configinstall "github.com/gardener/landscaper/apis/config/install"
	coreinstall "github.com/gardener/landscaper/apis/core/install"
)

// LandscaperScheme ist the scheme used in the landscaper cluster
var LandscaperScheme = runtime.NewScheme()

// ConfigScheme ist the scheme used for configurations
var ConfigScheme = runtime.NewScheme()

func init() {
	coreinstall.Install(LandscaperScheme)
	utilruntime.Must(corescheme.AddToScheme(LandscaperScheme))

	configinstall.Install(ConfigScheme)
}
