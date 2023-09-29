// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package aliases

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	Type   = cpi.AliasRepositoryType
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterRepositoryType(cpi.NewAliasRegistry(cpi.NewRepositoryType[*RepositorySpec](Type), setAlias))
	cpi.RegisterRepositoryType(cpi.NewRepositoryType[*RepositorySpec](TypeV1))
}

func setAlias(ctx cpi.Context, name string, spec cpi.RepositorySpec, creds cpi.CredentialsSource) error {
	r := ctx.GetAttributes().GetOrCreateAttribute(ATTR_REPOS, newRepositories)
	repos, ok := r.(*Repositories)
	if !ok {
		return fmt.Errorf("failed to assert type %T to Repositories", r)
	}
	repos.Set(name, spec, creds)
	return nil
}

// RepositorySpec describes a memory based repository interface.
type RepositorySpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	Alias                       string `json:"alias"`
}

// NewRepositorySpec creates a new memory RepositorySpec.
func NewRepositorySpec(name string) *RepositorySpec {
	return &RepositorySpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		Alias:               name,
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
	alias := repos.GetRepository(a.Alias)
	if alias == nil {
		return nil, cpi.ErrUnknownRepository(Type, a.Alias)
	}
	return alias.GetRepository(ctx, creds)
}
