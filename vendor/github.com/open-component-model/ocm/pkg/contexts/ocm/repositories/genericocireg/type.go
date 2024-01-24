// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"encoding/json"
	"path"

	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg/componentmapping"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

// ComponentNameMapping describes the method that is used to map the "Component Name", "Component Version"-tuples
// to OCI Image References.
type ComponentNameMapping string

const (
	Type = ocireg.Type

	OCIRegistryURLPathMapping ComponentNameMapping = "urlPath"
	OCIRegistryDigestMapping  ComponentNameMapping = "sha256-digest"
)

func init() {
	cpi.DefaultDelegationRegistry().Register("OCI", New(10))
}

// delegation tries to resolve an ocm repository specification
// with an OCI repository specification supported by the OCI context
// of the OCM context.
type delegation struct {
	prio int
}

func New(prio int) cpi.RepositoryPriorityDecoder {
	return &delegation{prio}
}

var _ cpi.RepositoryPriorityDecoder = (*delegation)(nil)

func (d *delegation) Decode(ctx cpi.Context, data []byte, unmarshal runtime.Unmarshaler) (cpi.RepositorySpec, error) {
	if unmarshal == nil {
		unmarshal = runtime.DefaultYAMLEncoding.Unmarshaler
	}

	ospec, err := ctx.OCIContext().RepositoryTypes().Decode(data, unmarshal)
	if err != nil {
		return nil, err
	}
	if oci.IsUnknown(ospec) {
		return nil, nil
	}

	meta := &ComponentRepositoryMeta{}
	err = unmarshal.Unmarshal(data, meta)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal component repository meta information")
	}
	return NewRepositorySpec(ospec, meta), nil
}

func (d *delegation) Priority() int {
	return d.prio
}

// ComponentRepositoryMeta describes config special for a mapping of
// a component repository to an oci registry.
// It is parsed in addition to an OCI based specification.
type ComponentRepositoryMeta struct {
	// ComponentNameMapping describes the method that is used to map the "Component Name", "Component Version"-tuples
	// to OCI Image References.
	ComponentNameMapping ComponentNameMapping `json:"componentNameMapping,omitempty"`
	SubPath              string               `json:"subPath,omitempty"`
}

func NewComponentRepositoryMeta(subPath string, mapping ComponentNameMapping) *ComponentRepositoryMeta {
	return DefaultComponentRepositoryMeta(&ComponentRepositoryMeta{
		ComponentNameMapping: mapping,
		SubPath:              subPath,
	})
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
	_ json.Marshaler                       = (*RepositorySpec)(nil)
	_ credentials.ConsumerIdentityProvider = (*RepositorySpec)(nil)
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

func (a *RepositorySpec) AsUniformSpec(cpi.Context) *cpi.UniformRepositorySpec {
	return &cpi.UniformRepositorySpec{Type: a.GetKind(), SubPath: a.SubPath}
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

// MarshalJSON implements a custom json unmarshal method for an unstructured type.
// The oci.RepositorySpec object might already implement json.Marshaler,
// which would be inherited and omit marshaling the addend attributes of a
// cpi.RepositorySpec.
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
	return NewRepository(ctx, &s.ComponentRepositoryMeta, r), nil
}

func (s *RepositorySpec) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	prefix := s.SubPath
	if c, ok := utils.Optional(uctx...).(credentials.StringUsageContext); ok {
		prefix = path.Join(prefix, componentmapping.ComponentDescriptorNamespace, c.String())
	}
	return credentials.GetProvidedConsumerId(s.RepositorySpec, credentials.StringUsageContext(prefix))
}

func (s *RepositorySpec) GetIdentityMatcher() string {
	return credentials.GetProvidedIdentityMatcher(s.RepositorySpec)
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
