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
	cpi.RegisterRepositoryType(Type, cpi.NewRepositoryType(Type, &RepositorySpec{}, nil))
	cpi.RegisterRepositoryType(TypeV1, cpi.NewRepositoryType(TypeV1, &RepositorySpec{}, nil))
}

type RepositorySpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	accessio.Options            `json:",inline"`

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
	o, err := accessio.AccessOptions(nil, opts...)
	if err != nil {
		return nil, err
	}
	return &RepositorySpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(Type),
		FilePath:            filePath,
		Options:             o,
		AccessMode:          acc,
	}, nil
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

func (a *RepositorySpec) AsUniformSpec(cpi.Context) cpi.UniformRepositorySpec {
	opts := &accessio.StandardOptions{}
	opts.Default()
	p, err := vfs.Canonical(opts.GetPathFileSystem(), a.FilePath, false)
	if err != nil {
		return cpi.UniformRepositorySpec{Type: a.GetKind(), SubPath: a.FilePath}
	}
	return cpi.UniformRepositorySpec{Type: a.GetKind(), SubPath: p}
}
