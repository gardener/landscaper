// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	Type   = "ComponentArchive"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterRepositoryType(cpi.NewRepositoryType[*RepositorySpec](Type, nil))
	cpi.RegisterRepositoryType(cpi.NewRepositoryType[*RepositorySpec](TypeV1, nil))
}

type RepositorySpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	accessio.StandardOptions    `json:",omitempty"`

	// FileFormat is the format of the repository file
	FilePath string `json:"filePath"`
	// AccessMode can be set to request readonly access or creation
	AccessMode accessobj.AccessMode `json:"accessMode,omitempty"`
}

var (
	_ accessio.Option                      = (*RepositorySpec)(nil)
	_ cpi.RepositorySpec                   = (*RepositorySpec)(nil)
	_ cpi.IntermediateRepositorySpecAspect = (*RepositorySpec)(nil)
)

// NewRepositorySpec creates a new RepositorySpec.
func NewRepositorySpec(acc accessobj.AccessMode, filePath string, opts ...accessio.Option) (*RepositorySpec, error) {
	spec := &RepositorySpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		FilePath:            filePath,
		AccessMode:          acc,
	}

	_, err := accessio.AccessOptions(&spec.StandardOptions, opts...)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func (a *RepositorySpec) IsIntermediate() bool {
	return true
}

func (a *RepositorySpec) GetType() string {
	return Type
}

func (a *RepositorySpec) Repository(ctx cpi.Context, creds credentials.Credentials) (cpi.Repository, error) {
	return NewRepository(ctx, a)
}

func (a *RepositorySpec) AsUniformSpec(cpi.Context) *cpi.UniformRepositorySpec {
	opts := &accessio.StandardOptions{}
	opts.Default()
	p, err := vfs.Canonical(opts.GetPathFileSystem(), a.FilePath, false)
	if err != nil {
		return &cpi.UniformRepositorySpec{Type: a.GetKind(), SubPath: a.FilePath}
	}
	return &cpi.UniformRepositorySpec{Type: a.GetKind(), SubPath: p}
}
