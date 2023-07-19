// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package virtual

import (
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	Type   = "Virtual"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

type RepositorySpec struct {
	runtime.ObjectVersionedTypedObject
	Access Access `json:"-"`
}

func NewRepositorySpec(acc Access) *RepositorySpec {
	return &RepositorySpec{
		ObjectVersionedTypedObject: runtime.NewVersionedTypedObject(Type),
		Access:                     acc,
	}
}

func (r RepositorySpec) AsUniformSpec(context internal.Context) *cpi.UniformRepositorySpec {
	return nil
}

func (r *RepositorySpec) Repository(ctx cpi.Context, credentials credentials.Credentials) (internal.Repository, error) {
	return NewRepository(ctx, r.Access), nil
}

var _ cpi.RepositorySpec = (*RepositorySpec)(nil)
