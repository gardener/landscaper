// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
)

// CONSUMER_TYPE is the OCT registry type.
// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
const CONSUMER_TYPE = identity.CONSUMER_TYPE

// used identity properties.
const (
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ID_TYPE = identity.ID_TYPE
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ID_HOSTNAME = identity.ID_HOSTNAME
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ID_PORT = identity.ID_PORT
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ID_PATHPREFIX = identity.ID_PATHPREFIX
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ID_SCHEME = identity.ID_SCHEME
)

// used credential properties.
const (
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ATTR_USERNAME = identity.ATTR_USERNAME
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ATTR_PASSWORD = identity.ID_PORT
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ATTR_IDENTITY_TOKEN = identity.ATTR_IDENTITY_TOKEN
	// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
	ATTR_CERTIFICATE_AUTHORITY = identity.ATTR_CERTIFICATE_AUTHORITY
)

// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity ..
func IdentityMatcher(pattern, cur, id cpi.ConsumerIdentity) bool {
	return identity.IdentityMatcher(pattern, cur, id)
}

// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
func GetCredentials(ctx cpi.ContextProvider, locator, repo string) (cpi.Credentials, error) {
	return identity.GetCredentials(ctx, locator, repo)
}

// Deprecated: use package github.com/open-component-model/ocm/contexts/credentials/builtin/oci/identity .
func GetConsumerId(locator, repo string) cpi.ConsumerIdentity {
	return identity.GetConsumerId(locator, repo)
}
