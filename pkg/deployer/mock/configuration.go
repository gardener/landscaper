// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockinstall "github.com/gardener/landscaper/apis/deployer/mock/install"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/utils"
)

// Type is the type name of the deployer.
const Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/mock"

const Name = "mock.deployer.landscaper.gardener.cloud"

var (
	MockScheme = runtime.NewScheme()
	Decoder    runtime.Decoder
)

func init() {
	mockinstall.Install(MockScheme)
	Decoder = api.NewDecoder(MockScheme)
}

// NewDeployItemBuilder creates a new deployitem builder for mock deployitems
func NewDeployItemBuilder() *utils.DeployItemBuilder {
	return utils.NewDeployItemBuilder(string(Type)).Scheme(MockScheme)
}
