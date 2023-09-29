// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"path"
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/identity/hostpath"
	"github.com/open-component-model/ocm/pkg/listformat"
)

const CONSUMER_TYPE = "S3"

// identity properties.
const (
	ID_HOSTNAME   = hostpath.ID_HOSTNAME
	ID_PORT       = hostpath.ID_PORT
	ID_PATHPREFIX = hostpath.ID_PATHPREFIX
)

// credential properties.
const (
	ATTR_AWS_ACCESS_KEY_ID     = "awsAccessKeyID"
	ATTR_AWS_SECRET_ACCESS_KEY = "awsSecretAccessKey"
	ATTR_TOKEN                 = cpi.ATTR_TOKEN
)

const GITHUB = "github.com"

var identityMatcher = hostpath.IdentityMatcher(CONSUMER_TYPE)

func IdentityMatcher(pattern, cur, id cpi.ConsumerIdentity) bool {
	return identityMatcher(pattern, cur, id)
}

func init() {
	attrs := listformat.FormatListElements("", listformat.StringElementDescriptionList{
		ATTR_AWS_ACCESS_KEY_ID, "AWS access key id",
		ATTR_AWS_SECRET_ACCESS_KEY, "AWS secret for access key id",
		ATTR_TOKEN, "AWS access token (alternatively)",
	})
	cpi.RegisterStandardIdentity(CONSUMER_TYPE, identityMatcher,
		`S3 credential matcher

This matcher is a hostpath matcher.`,
		attrs)
}

func GetConsumerId(host, bucket, key, version string) cpi.ConsumerIdentity {
	id := cpi.NewConsumerIdentity(CONSUMER_TYPE)

	parts := strings.Split(host, ":")
	if parts[0] != "" {
		id[ID_HOSTNAME] = parts[0]
	}
	if len(parts) > 1 {
		id[ID_PORT] = parts[1]
	}
	id[ID_PATHPREFIX] = path.Join(bucket, key, version)
	return id
}

func GetCredentials(ctx cpi.ContextProvider, host, bucket, key, version string) (cpi.Credentials, error) {
	id := GetConsumerId(host, bucket, key, version)
	return cpi.CredentialsForConsumer(ctx.CredentialsContext(), id, identityMatcher)
}
