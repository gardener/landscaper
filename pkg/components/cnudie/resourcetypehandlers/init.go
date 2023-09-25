// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package resourcetypehandlers

import (
	_ "github.com/gardener/landscaper/pkg/components/cnudie/resourcetypehandlers/blueprint"
	_ "github.com/gardener/landscaper/pkg/components/cnudie/resourcetypehandlers/helmchart"
	_ "github.com/gardener/landscaper/pkg/components/cnudie/resourcetypehandlers/jsonschema"
)
