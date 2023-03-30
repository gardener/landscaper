// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package directcreds

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	Type   = "Credentials"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterRepositoryType(Type, cpi.NewRepositoryType(Type, &RepositorySpec{}))
	cpi.RegisterRepositoryType(TypeV1, cpi.NewRepositoryType(TypeV1, &RepositorySpec{}))
}

// RepositorySpec describes a repository interface for single direct credentials.
type RepositorySpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	Properties                  common.Properties `json:"properties"`
}

var (
	_ cpi.RepositorySpec  = &RepositorySpec{}
	_ cpi.CredentialsSpec = &RepositorySpec{}
)

// NewRepositorySpec creates a new RepositorySpec.
func NewRepositorySpec(credentials common.Properties) *RepositorySpec {
	return &RepositorySpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(Type),
		Properties:          credentials,
	}
}

func (a *RepositorySpec) GetType() string {
	return Type
}

func (a *RepositorySpec) Repository(ctx cpi.Context, creds cpi.Credentials) (cpi.Repository, error) {
	return NewRepository(cpi.NewCredentials(a.Properties)), nil
}

func (a *RepositorySpec) Credentials(context cpi.Context, source ...cpi.CredentialsSource) (cpi.Credentials, error) {
	return cpi.NewCredentials(a.Properties), nil
}

func (a *RepositorySpec) GetCredentialsName() string {
	return ""
}

func (a *RepositorySpec) GetRepositorySpec(context cpi.Context) cpi.RepositorySpec {
	return a
}
