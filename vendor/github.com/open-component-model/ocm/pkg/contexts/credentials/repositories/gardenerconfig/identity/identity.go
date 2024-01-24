// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/identity/hostpath"
	"github.com/open-component-model/ocm/pkg/listformat"
	"github.com/open-component-model/ocm/pkg/utils"
)

const CONSUMER_TYPE = "Buildcredentials" + common.OCM_TYPE_GROUP_SUFFIX

// used identity attributes.
const (
	ID_SCHEME     = hostpath.ID_SCHEME
	ID_HOSTNAME   = hostpath.ID_HOSTNAME
	ID_PORT       = hostpath.ID_PORT
	ID_PATHPREFIX = hostpath.ID_PATHPREFIX
)

// used credential properties.
const (
	ATTR_KEY = cpi.ATTR_KEY
)

var identityMatcher = hostpath.IdentityMatcher(CONSUMER_TYPE)

func IdentityMatcher(pattern, cur, id cpi.ConsumerIdentity) bool {
	return identityMatcher(pattern, cur, id)
}

func init() {
	attrs := listformat.FormatListElements("", listformat.StringElementDescriptionList{
		ATTR_KEY, "secret key use to access the credential server",
	})

	cpi.RegisterStandardIdentity(CONSUMER_TYPE, IdentityMatcher, `Gardener config credential matcher

It matches the <code>`+CONSUMER_TYPE+`</code> consumer type and additionally acts like
the <code>`+hostpath.IDENTITY_TYPE+`</code> type.`,
		attrs)
}

func GetConsumerId(configURL string) (cpi.ConsumerIdentity, error) {
	parsedURL, err := utils.ParseURL(configURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url: %w", err)
	}

	id := cpi.NewConsumerIdentity(CONSUMER_TYPE)
	id.SetNonEmptyValue(ID_HOSTNAME, parsedURL.Host)
	id.SetNonEmptyValue(ID_SCHEME, parsedURL.Scheme)
	id.SetNonEmptyValue(ID_PATHPREFIX, strings.Trim(parsedURL.Path, "/"))
	id.SetNonEmptyValue(ID_PORT, parsedURL.Port())

	return id, nil
}
