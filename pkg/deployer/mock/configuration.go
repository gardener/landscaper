// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockinstall "github.com/gardener/landscaper/apis/deployer/mock/install"
)

// Type is the type name of the deployer.
const Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/mock"

var Mockscheme = runtime.NewScheme()

func init() {
	mockinstall.Install(Mockscheme)
}
