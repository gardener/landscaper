// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gardenerconfig

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/identity/hostpath"
	gardenercfgcpi "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/gardenerconfig/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	RepositoryType   = "GardenerConfig"
	RepositoryTypeV1 = RepositoryType + runtime.VersionSeparator + "v1"
	CONSUMER_TYPE    = "Buildcredentials" + common.OCM_TYPE_GROUP_SUFFIX
)

var identityMatcher = hostpath.IdentityMatcher(CONSUMER_TYPE)

func IdentityMatcher(pattern, cur, id cpi.ConsumerIdentity) bool {
	return identityMatcher(pattern, cur, id)
}

func init() {
	cpi.RegisterRepositoryType(RepositoryType, cpi.NewRepositoryType(RepositoryType, &RepositorySpec{}))
	cpi.RegisterRepositoryType(RepositoryTypeV1, cpi.NewRepositoryType(RepositoryTypeV1, &RepositorySpec{}))

	cpi.RegisterStandardIdentityMatcher(CONSUMER_TYPE, IdentityMatcher, `Gardener config credential matcher

It matches the <code>`+CONSUMER_TYPE+`</code> consumer type and additionally acts like
the <code>`+hostpath.IDENTITY_TYPE+`</code> type.`)
}

// RepositorySpec describes a secret server based credential repository interface.
type RepositorySpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	URL                         string                    `json:"url"`
	ConfigType                  gardenercfgcpi.ConfigType `json:"configType"`
	Cipher                      Cipher                    `json:"cipher"`
	PropagateConsumerIdentity   bool                      `json:"propagateConsumerIdentity"`
}

// NewRepositorySpec creates a new memory RepositorySpec.
func NewRepositorySpec(url string, configType gardenercfgcpi.ConfigType, cipher Cipher, propagateConsumerIdentity bool) *RepositorySpec {
	return &RepositorySpec{
		ObjectVersionedType:       runtime.NewVersionedObjectType(RepositoryType),
		URL:                       url,
		ConfigType:                configType,
		Cipher:                    cipher,
		PropagateConsumerIdentity: propagateConsumerIdentity,
	}
}

func (a *RepositorySpec) GetType() string {
	return RepositoryType
}

func (a *RepositorySpec) Repository(ctx cpi.Context, creds cpi.Credentials) (cpi.Repository, error) {
	r := ctx.GetAttributes().GetOrCreateAttribute(ATTR_REPOS, newRepositories)
	repos, ok := r.(*Repositories)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to Responsitories", r)
	}

	key, err := getKey(ctx, a.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to get key from context: %w", err)
	}

	return repos.GetRepository(ctx, a.URL, a.ConfigType, a.Cipher, key, a.PropagateConsumerIdentity)
}

func getKey(cctx cpi.Context, configURL string) ([]byte, error) {
	parsedURL, err := utils.ParseURL(configURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url: %w", err)
	}

	id := cpi.ConsumerIdentity{
		cpi.ID_TYPE: CONSUMER_TYPE,
	}
	id.SetNonEmptyValue(hostpath.ID_HOSTNAME, parsedURL.Host)
	id.SetNonEmptyValue(hostpath.ID_SCHEME, parsedURL.Scheme)
	id.SetNonEmptyValue(hostpath.ID_PATHPREFIX, strings.Trim(parsedURL.Path, "/"))
	id.SetNonEmptyValue(hostpath.ID_PORT, parsedURL.Port())

	creds, err := cpi.CredentialsForConsumer(cctx, id, identityMatcher)
	if err != nil {
		return nil, err
	}

	var key string
	if creds != nil {
		key = creds.GetProperty(cpi.ATTR_KEY)
	}

	return []byte(key), nil
}
