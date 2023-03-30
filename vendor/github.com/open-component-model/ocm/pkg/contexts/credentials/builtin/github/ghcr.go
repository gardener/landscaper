// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"os"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
)

const HOST = "ghcr.io"

func init() {
	t := os.Getenv("GITHUB_TOKEN")
	if t != "" {
		host := os.Getenv("GITHUB_HOST")
		if host == "" {
			host = HOST
		}
		id := cpi.ConsumerIdentity{
			identity.ID_TYPE:     identity.CONSUMER_TYPE,
			identity.ID_HOSTNAME: host,
		}
		if src, err := cpi.DefaultContext.GetCredentialsForConsumer(id); err != nil || src == nil {
			creds := cpi.NewCredentials(common.Properties{cpi.ATTR_IDENTITY_TOKEN: t})
			cpi.DefaultContext.SetCredentialsForConsumer(id, creds)
		}
	}
}
