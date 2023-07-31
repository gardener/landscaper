// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"strings"

	"helm.sh/helm/v3/pkg/registry"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/identity/hostpath"
	ociidentity "github.com/open-component-model/ocm/pkg/contexts/oci/identity"
)

// CONSUMER_TYPE is the Helm chart repository type.
const CONSUMER_TYPE = "HelmChartRepository"

// ID_TYPE is the type field of a consumer identity.
const ID_TYPE = cpi.ID_TYPE

// ID_SCHEME is the scheme of the repository.
const ID_SCHEME = hostpath.ID_SCHEME

// ID_HOSTNAME is the hostname of a repository.
const ID_HOSTNAME = hostpath.ID_HOSTNAME

// ID_PORT is the port number of a repository.
const ID_PORT = hostpath.ID_PORT

// ID_PATHPREFIX is the path of a repository.
const ID_PATHPREFIX = hostpath.ID_PATHPREFIX

func init() {
	cpi.RegisterStandardIdentity(CONSUMER_TYPE, IdentityMatcher, `Helm chart repository

It matches the <code>`+CONSUMER_TYPE+`</code> consumer type and additionally acts like 
the <code>`+hostpath.IDENTITY_TYPE+`</code> type.`,
		`- **<code>`+ATTR_USERNAME+`</code>**: basic auth user name.
- **<code>`+ATTR_PASSWORD+`</code>**: basic auth password.
- **<code>`+ATTR_CERTIFICATE+`</code>**: TLS client certificate.
- **<code>`+ATTR_PRIVATE_KEY+`</code>**: TLS private key.`)
}

var identityMatcher = hostpath.IdentityMatcher(CONSUMER_TYPE)

func IdentityMatcher(pattern, cur, id cpi.ConsumerIdentity) bool {
	return identityMatcher(pattern, cur, id)
}

// used credential attributes

const (
	ATTR_USERNAME              = credentials.ATTR_USERNAME
	ATTR_PASSWORD              = credentials.ATTR_PASSWORD
	ATTR_CERTIFICATE_AUTHORITY = credentials.ATTR_CERTIFICATE_AUTHORITY
	ATTR_CERTIFICATE           = credentials.ATTR_CERTIFICATE
	ATTR_PRIVATE_KEY           = credentials.ATTR_PRIVATE_KEY
)

func OCIRepoURL(repourl string, chartname string) string {
	repourl = strings.TrimSuffix(repourl, "/")[3+len(registry.OCIScheme):]
	if chartname != "" {
		repourl += "/" + chartname
	}
	return repourl
}

func GetConsumerId(repourl string, chartname string) cpi.ConsumerIdentity {
	if registry.IsOCI(repourl) {
		repourl = strings.TrimSuffix(repourl, "/")
		return ociidentity.GetConsumerId(OCIRepoURL(repourl, ""), chartname)
	} else {
		return hostpath.GetConsumerIdentity(CONSUMER_TYPE, repourl)
	}
}

func GetCredentials(ctx credentials.ContextProvider, repourl string, chartname string) common.Properties {
	id := GetConsumerId(repourl, chartname)
	if id == nil {
		return nil
	}
	creds, err := credentials.CredentialsForConsumer(ctx.CredentialsContext(), id, identityMatcher)
	if creds == nil || err != nil {
		return nil
	}
	return creds.Properties()
}
