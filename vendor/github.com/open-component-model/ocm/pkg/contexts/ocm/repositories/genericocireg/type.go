// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// ComponentNameMapping describes the method that is used to map the "Component Name", "Component Version"-tuples
// to OCI Image References.
type ComponentNameMapping string

const (
	Type   = ocireg.Type
	TypeV1 = ocireg.TypeV1

	OCIRegistryURLPathMapping ComponentNameMapping = "urlPath"
	OCIRegistryDigestMapping  ComponentNameMapping = "sha256-digest"
)

func init() {
	cpi.RegisterOCIImplementation(func(ctx oci.Context) (cpi.RepositoryType, error) {
		return NewRepositoryType(ctx), nil
	})
}

// ComponentRepositoryMeta describes config special for a mapping of
// a component repository to an oci registry.
type ComponentRepositoryMeta struct {
	// ComponentNameMapping describes the method that is used to map the "Component Name", "Component Version"-tuples
	// to OCI Image References.
	ComponentNameMapping ComponentNameMapping `json:"componentNameMapping,omitempty"`
	SubPath              string               `json:"subPath,omitempty"`
}

func NewComponentRepositoryMeta(subPath string, mapping ComponentNameMapping) *ComponentRepositoryMeta {
	return &ComponentRepositoryMeta{
		ComponentNameMapping: mapping,
		SubPath:              subPath,
	}
}

type RepositoryType struct {
	runtime.ObjectVersionedType
	ocictx oci.Context
}

var _ cpi.RepositoryType = &RepositoryType{}

// NewRepositoryType creates generic type for any OCI Repository Backend.
func NewRepositoryType(ocictx oci.Context) *RepositoryType {
	return &RepositoryType{
		ObjectVersionedType: runtime.NewVersionedObjectType("genericOCIRepositoryBackend"),
		ocictx:              ocictx,
	}
}

func (t *RepositoryType) Decode(data []byte, unmarshal runtime.Unmarshaler) (runtime.TypedObject, error) {
	ospec, err := t.ocictx.RepositoryTypes().DecodeRepositorySpec(data, unmarshal)
	if err != nil {
		return nil, err
	}

	meta := &ComponentRepositoryMeta{}
	if unmarshal == nil {
		unmarshal = runtime.DefaultYAMLEncoding.Unmarshaler
	}
	err = unmarshal.Unmarshal(data, meta)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal component repository meta information")
	}
	return NewRepositorySpec(ospec, meta), nil
}

func (t *RepositoryType) LocalSupportForAccessSpec(ctx cpi.Context, a compdesc.AccessSpec) bool {
	name := a.GetKind()
	return name == localblob.Type
}

////////////////////////////////////////////////////////////////////////////////

type RepositorySpec struct {
	oci.RepositorySpec
	ComponentRepositoryMeta
}

var (
	_ cpi.RepositorySpec                   = (*RepositorySpec)(nil)
	_ cpi.PrefixProvider                   = (*RepositorySpec)(nil)
	_ cpi.IntermediateRepositorySpecAspect = (*RepositorySpec)(nil)
)

func NewRepositorySpec(spec oci.RepositorySpec, meta *ComponentRepositoryMeta) *RepositorySpec {
	return &RepositorySpec{
		RepositorySpec:          spec,
		ComponentRepositoryMeta: *DefaultComponentRepositoryMeta(meta),
	}
}

func (a *RepositorySpec) PathPrefix() string {
	return a.SubPath
}

func (a *RepositorySpec) IsIntermediate() bool {
	if s, ok := a.RepositorySpec.(oci.IntermediateRepositorySpecAspect); ok {
		return s.IsIntermediate()
	}
	return false
}

// TODO: host etc is missing

func (a *RepositorySpec) AsUniformSpec(cpi.Context) cpi.UniformRepositorySpec {
	return cpi.UniformRepositorySpec{Type: a.GetKind(), SubPath: a.SubPath}
}

func (u *RepositorySpec) UnmarshalJSON(data []byte) error {
	logrus.Debugf("unmarshal generic ocireg spec %s\n", string(data))
	ocispec := &oci.GenericRepositorySpec{}
	if err := json.Unmarshal(data, ocispec); err != nil {
		return err
	}
	compmeta := &ComponentRepositoryMeta{}
	if err := json.Unmarshal(data, ocispec); err != nil {
		return err
	}

	u.RepositorySpec = ocispec
	u.ComponentRepositoryMeta = *compmeta
	return nil
}

// MarshalJSON implements a custom json unmarshal method for a unstructured type.
func (u RepositorySpec) MarshalJSON() ([]byte, error) {
	ocispec, err := runtime.ToUnstructuredTypedObject(u.RepositorySpec)
	if err != nil {
		return nil, err
	}
	compmeta, err := runtime.ToUnstructuredObject(u.ComponentRepositoryMeta)
	if err != nil {
		return nil, err
	}
	return json.Marshal(compmeta.FlatMerge(ocispec.Object))
}

func (s *RepositorySpec) Repository(ctx cpi.Context, creds credentials.Credentials) (cpi.Repository, error) {
	r, err := s.RepositorySpec.Repository(ctx.OCIContext(), creds)
	if err != nil {
		return nil, err
	}
	return NewRepository(ctx, &s.ComponentRepositoryMeta, r)
}

func DefaultComponentRepositoryMeta(meta *ComponentRepositoryMeta) *ComponentRepositoryMeta {
	if meta == nil {
		meta = &ComponentRepositoryMeta{}
	}
	if meta.ComponentNameMapping == "" {
		meta.ComponentNameMapping = OCIRegistryURLPathMapping
	}
	return meta
}
