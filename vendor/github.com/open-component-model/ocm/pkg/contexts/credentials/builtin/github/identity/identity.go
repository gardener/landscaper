// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"net/url"
	"path"
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/identity/hostpath"
	"github.com/open-component-model/ocm/pkg/listformat"
)

const CONSUMER_TYPE = "Github"

// identity properties
const (
	ID_HOSTNAME   = hostpath.ID_HOSTNAME
	ID_PORT       = hostpath.ID_PORT
	ID_PATHPREFIX = hostpath.ID_PATHPREFIX
)

// credential properties
const (
	ATTR_TOKEN = cpi.ATTR_TOKEN
)

const GITHUB = "github.com"

var identityMatcher = hostpath.IdentityMatcher(CONSUMER_TYPE)

func IdentityMatcher(pattern, cur, id cpi.ConsumerIdentity) bool {
	return identityMatcher(pattern, cur, id)
}

func init() {
	attrs := listformat.FormatListElements("", listformat.StringElementDescriptionList{
		ATTR_TOKEN, "GitHub personal access token",
	})
	cpi.RegisterStandardIdentity(CONSUMER_TYPE, identityMatcher,
		`GitHub credential matcher

This matcher is a hostpath matcher.`,
		attrs)
}

func PATCredentials(pat string) cpi.Credentials {
	return cpi.DirectCredentials{
		ATTR_TOKEN: pat,
	}
}

func GetConsumerId(serverurl string, repo ...string) cpi.ConsumerIdentity {
	host := GITHUB
	port := ""
	if serverurl != "" {
		u, err := url.Parse(serverurl)
		if err != nil {
			host = u.Host
		}
	}
	if idx := strings.Index(host, ":"); idx > 0 {
		port = host[idx+1:]
		host = host[:idx]
	}

	id := cpi.ConsumerIdentity{
		cpi.ID_TYPE: CONSUMER_TYPE,
		ID_HOSTNAME: host,
	}
	if port != "" {
		id[ID_PORT] = port
	}
	p := path.Join(repo...)
	if p != "" {
		id[ID_PATHPREFIX] = p
	}
	return id
}

func GetCredentials(ctx cpi.ContextProvider, serverurl string, repo ...string) (cpi.Credentials, error) {
	id := GetConsumerId(serverurl, repo...)
	return cpi.CredentialsForConsumer(ctx.CredentialsContext(), id, identityMatcher)
}
