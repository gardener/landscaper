// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
)

var (
	ocmFactory model.Factory = &ocmlib.Factory{}
)

func GetFactory() model.Factory {
	return ocmFactory
}
