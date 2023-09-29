// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"github.com/open-component-model/ocm/pkg/logging"
)

var (
	REALM  = logging.DefineSubRealm("component descriptor handling", "compdesc")
	Logger = logging.DynamicLogger(REALM)
)
