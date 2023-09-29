// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package directcreds

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
)

func NewCredentials(props common.Properties) cpi.CredentialsSpec {
	return cpi.NewCredentialsSpec(Type, NewRepositorySpec(props))
}
