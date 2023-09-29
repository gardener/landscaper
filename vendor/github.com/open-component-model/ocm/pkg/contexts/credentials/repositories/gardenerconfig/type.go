// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gardenerconfig

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	gardenercfgcpi "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/gardenerconfig/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/gardenerconfig/identity"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	Type   = "GardenerConfig"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterRepositoryType(cpi.NewRepositoryType[*RepositorySpec](Type))
	cpi.RegisterRepositoryType(cpi.NewRepositoryType[*RepositorySpec](TypeV1))
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
		ObjectVersionedType:       runtime.NewVersionedTypedObject(Type),
		URL:                       url,
		ConfigType:                configType,
		Cipher:                    cipher,
		PropagateConsumerIdentity: propagateConsumerIdentity,
	}
}

func (a *RepositorySpec) GetType() string {
	return Type
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

	id := cpi.NewConsumerIdentity(identity.CONSUMER_TYPE)
	id.SetNonEmptyValue(identity.ID_HOSTNAME, parsedURL.Host)
	id.SetNonEmptyValue(identity.ID_SCHEME, parsedURL.Scheme)
	id.SetNonEmptyValue(identity.ID_PATHPREFIX, strings.Trim(parsedURL.Path, "/"))
	id.SetNonEmptyValue(identity.ID_PORT, parsedURL.Port())

	creds, err := cpi.CredentialsForConsumer(cctx, id)
	if err != nil {
		return nil, err
	}

	var key string
	if creds != nil {
		key = creds.GetProperty(identity.ATTR_KEY)
	}

	return []byte(key), nil
}
