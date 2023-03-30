// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"strings"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	Type   = cpi.CommonTransportFormat
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterRepositoryType(Type, cpi.NewRepositoryType(Type, &RepositorySpec{}))
	cpi.RegisterRepositoryType(TypeV1, cpi.NewRepositoryType(TypeV1, &RepositorySpec{}))
}

// RepositorySpec describes an OCI registry interface backed by an oci registry.
type RepositorySpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	accessio.StandardOptions    `json:",inline"`

	// FileFormat is the format of the repository file
	FilePath string `json:"filePath"`
	// AccessMode can be set to request readonly access or creation
	AccessMode accessobj.AccessMode `json:"accessMode,omitempty"`
}

var _ cpi.RepositorySpec = (*RepositorySpec)(nil)

var _ cpi.IntermediateRepositorySpecAspect = (*RepositorySpec)(nil)

// NewRepositorySpec creates a new RepositorySpec.
func NewRepositorySpec(mode accessobj.AccessMode, filePath string, opts ...accessio.Option) (*RepositorySpec, error) {
	o, err := accessio.AccessOptions(nil, opts...)
	if err != nil {
		return nil, err
	}
	if o.GetFileFormat() == nil {
		for _, v := range SupportedFormats() {
			if strings.HasSuffix(filePath, "."+v.String()) {
				o.SetFileFormat(v)
				break
			}
		}
	}
	o.Default()
	return &RepositorySpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(Type),
		FilePath:            filePath,
		StandardOptions:     *o.(*accessio.StandardOptions),
		AccessMode:          mode,
	}, nil
}

func (a *RepositorySpec) IsIntermediate() bool {
	return true
}

func (a *RepositorySpec) GetType() string {
	return Type
}

func (s *RepositorySpec) Name() string {
	return s.FilePath
}

func (s *RepositorySpec) UniformRepositorySpec() *cpi.UniformRepositorySpec {
	u := &cpi.UniformRepositorySpec{
		Type: Type,
		Info: s.FilePath,
	}
	return u
}

func (a *RepositorySpec) Repository(ctx cpi.Context, creds credentials.Credentials) (cpi.Repository, error) {
	return Open(ctx, a.AccessMode, a.FilePath, 0o700, &a.StandardOptions)
}
