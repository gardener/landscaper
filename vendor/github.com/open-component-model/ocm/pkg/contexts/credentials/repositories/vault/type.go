// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"encoding/json"
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/optionutils"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	Type   = "HashiCorpVault"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterRepositoryType(cpi.NewRepositoryType[*RepositorySpec](Type))
	cpi.RegisterRepositoryType(cpi.NewRepositoryType[*RepositorySpec](TypeV1, cpi.WithDescription(usage), cpi.WithFormatSpec(format)))
}

// RepositorySpec describes a docker config based credential repository interface.
type RepositorySpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	ServerURL                   string `json:"serverURL"`
	Options                     `json:",inline"`
}

// NewRepositorySpec creates a new memory RepositorySpec.
func NewRepositorySpec(url string, opts ...Option) *RepositorySpec {
	return &RepositorySpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		ServerURL:           url,
		Options:             *optionutils.EvalOptions(opts...),
	}
}

func (a *RepositorySpec) GetType() string {
	return Type
}

func (a *RepositorySpec) Repository(ctx cpi.Context, creds cpi.Credentials) (cpi.Repository, error) {
	r := ctx.GetAttributes().GetOrCreateAttribute(ATTR_REPOS, newRepositories)
	repos, ok := r.(*Repositories)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to Repositories", r)
	}
	spec := *a
	spec.Secrets = slices.Clone(a.Secrets)
	if spec.SecretsEngine == "" {
		spec.SecretsEngine = "secrets"
	}
	return repos.GetRepository(ctx, &spec)
}

func (a *RepositorySpec) GetKey() cpi.ProviderIdentity {
	spec := *a
	spec.PropgateConsumerIdentity = false
	data, err := json.Marshal(&spec)
	if err == nil {
		return cpi.ProviderIdentity(data)
	}
	return cpi.ProviderIdentity(spec.ServerURL)
}
