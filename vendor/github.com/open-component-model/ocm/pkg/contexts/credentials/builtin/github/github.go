// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"os"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/github/identity"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
)

func init() {
	t := os.Getenv("GITHUB_TOKEN")
	if t != "" {
		us := os.Getenv("GITHUB_SERVER_URL")
		id := identity.GetConsumerId(us)

		if src, err := cpi.DefaultContext.GetCredentialsForConsumer(id); err != nil || src == nil {
			creds := cpi.NewCredentials(common.Properties{cpi.ATTR_TOKEN: t})
			cpi.DefaultContext.SetCredentialsForConsumer(id, creds)
		}
	}
}
