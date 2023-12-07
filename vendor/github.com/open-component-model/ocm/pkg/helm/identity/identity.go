// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	helmidentity "github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity"
)

// CONSUMER_TYPE is the Helm chart repository type.
// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
const CONSUMER_TYPE = helmidentity.CONSUMER_TYPE

// ID_TYPE is the type field of a consumer identity.
// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
const ID_TYPE = helmidentity.ID_PORT

// ID_SCHEME is the scheme of the repository.
// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
const ID_SCHEME = helmidentity.ID_SCHEME

// ID_HOSTNAME is the hostname of a repository.
// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
const ID_HOSTNAME = helmidentity.ID_HOSTNAME

// ID_PORT is the port number of a repository.
// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
const ID_PORT = helmidentity.ID_PORT

// ID_PATHPREFIX is the path of a repository.
// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
const ID_PATHPREFIX = helmidentity.ID_PATHPREFIX

// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
func IdentityMatcher(pattern, cur, id credentials.ConsumerIdentity) bool {
	return helmidentity.IdentityMatcher(pattern, cur, id)
}

// used credential attributes

const (
	// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
	ATTR_USERNAME = credentials.ATTR_USERNAME
	// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
	ATTR_PASSWORD = credentials.ATTR_PASSWORD
	// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
	ATTR_CERTIFICATE_AUTHORITY = credentials.ATTR_CERTIFICATE_AUTHORITY
	// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
	ATTR_CERTIFICATE = credentials.ATTR_CERTIFICATE
	// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
	ATTR_PRIVATE_KEY = credentials.ATTR_PRIVATE_KEY
)

// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
func OCIRepoURL(repourl string, chartname string) string {
	return helmidentity.OCIRepoURL(repourl, chartname)
}

// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
func SimpleCredentials(user, passwd string) credentials.Credentials {
	return helmidentity.SimpleCredentials(user, passwd)
}

// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
func GetConsumerId(repourl string, chartname string) credentials.ConsumerIdentity {
	return helmidentity.GetConsumerId(repourl, chartname)
}

// Deprecated: use package github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity.
func GetCredentials(ctx credentials.ContextProvider, repourl string, chartname string) common.Properties {
	return helmidentity.GetCredentials(ctx, repourl, chartname)
}
