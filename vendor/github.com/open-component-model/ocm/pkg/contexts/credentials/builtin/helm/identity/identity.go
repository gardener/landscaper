// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"strings"

	"helm.sh/helm/v3/pkg/registry"

	"github.com/open-component-model/ocm/pkg/common"
	ociidentity "github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/identity/hostpath"
	"github.com/open-component-model/ocm/pkg/listformat"
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
	attrs := listformat.FormatListElements("", listformat.StringElementDescriptionList{
		ATTR_USERNAME, "the basic auth user name",
		ATTR_PASSWORD, "the basic auth password",
		ATTR_CERTIFICATE, "TLS client certificate",
		ATTR_PRIVATE_KEY, "TLS private key",
		ATTR_CERTIFICATE_AUTHORITY, "TLS certificate authority",
	})

	cpi.RegisterStandardIdentity(CONSUMER_TYPE, IdentityMatcher, `Helm chart repository

It matches the <code>`+CONSUMER_TYPE+`</code> consumer type and additionally acts like 
the <code>`+hostpath.IDENTITY_TYPE+`</code> type.`,
		attrs)
}

var identityMatcher = hostpath.IdentityMatcher("")

func IdentityMatcher(pattern, cur, id cpi.ConsumerIdentity) bool {
	return identityMatcher(pattern, cur, id)
}

// used credential attributes

const (
	ATTR_USERNAME              = cpi.ATTR_USERNAME
	ATTR_PASSWORD              = cpi.ATTR_PASSWORD
	ATTR_CERTIFICATE_AUTHORITY = cpi.ATTR_CERTIFICATE_AUTHORITY
	ATTR_CERTIFICATE           = cpi.ATTR_CERTIFICATE
	ATTR_PRIVATE_KEY           = cpi.ATTR_PRIVATE_KEY
)

func OCIRepoURL(repourl string, chartname string) string {
	repourl = strings.TrimSuffix(repourl, "/")[3+len(registry.OCIScheme):]
	if chartname != "" {
		repourl += "/" + chartname
	}
	return repourl
}

func SimpleCredentials(user, passwd string) cpi.Credentials {
	return cpi.DirectCredentials{
		ATTR_USERNAME: user,
		ATTR_PASSWORD: passwd,
	}
}

func GetConsumerId(repourl string, chartname string) cpi.ConsumerIdentity {
	i := strings.LastIndex(chartname, ":")
	if i >= 0 {
		chartname = chartname[:i]
	}
	if registry.IsOCI(repourl) {
		repourl = strings.TrimSuffix(repourl, "/")
		return ociidentity.GetConsumerId(OCIRepoURL(repourl, ""), chartname)
	} else {
		return hostpath.GetConsumerIdentity(CONSUMER_TYPE, repourl)
	}
}

func GetCredentials(ctx cpi.ContextProvider, repourl string, chartname string) common.Properties {
	id := GetConsumerId(repourl, chartname)
	if id == nil {
		return nil
	}
	creds, err := cpi.CredentialsForConsumer(ctx.CredentialsContext(), id)
	if creds == nil || err != nil {
		return nil
	}
	return creds.Properties()
}
