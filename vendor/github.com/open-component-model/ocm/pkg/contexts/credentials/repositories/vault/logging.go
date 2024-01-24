// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	ocmlog "github.com/open-component-model/ocm/pkg/logging"
)

var (
	REALM = ocmlog.DefineSubRealm("HashiCorp Vault Access", "credentials", "vault")
	log   = ocmlog.DynamicLogger(REALM)
)
