// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociartifact

import (
	"fmt"
	"io"
	"strings"
	"sync"

	. "github.com/open-component-model/ocm/pkg/finalizer"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type of a oci registry.
const (
	Type   = "ociArtifact"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

const (
	LegacyType   = "ociRegistry"
	LegacyTypeV1 = LegacyType + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](Type, cpi.WithDescription(usage)))
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](TypeV1, cpi.WithFormatSpec(formatV1), cpi.WithConfigHandler(ConfigHandler())))

	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](LegacyType))
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](LegacyTypeV1))
}

func Is(spec cpi.AccessSpec) bool {
	return spec != nil && spec.GetKind() == Type || spec.GetKind() == LegacyType
}

// AccessSpec describes the access for a oci registry.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`

	// ImageReference is the actual reference to the oci image repository and tag.
	ImageReference string `json:"imageReference"`
}

var (
	_ cpi.AccessSpec   = (*AccessSpec)(nil)
	_ cpi.HintProvider = (*AccessSpec)(nil)
)

// New creates a new oci registry access spec version v1.
func New(ref string) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		ImageReference:      ref,
	}
}

func (a *AccessSpec) Describe(ctx cpi.Context) string {
	return fmt.Sprintf("OCI artifact %s", a.ImageReference)
}

func (_ *AccessSpec) IsLocal(cpi.Context) bool {
	return false
}

func (a *AccessSpec) GlobalAccessSpec(ctx cpi.Context) cpi.AccessSpec {
	return a
}

func (a *AccessSpec) GetReferenceHint(cv cpi.ComponentVersionAccess) string {
	ref, err := oci.ParseRef(a.ImageReference)
	if err != nil {
		return ""
	}
	prefix := cpi.RepositoryPrefix(cv.Repository().GetSpecification())
	hint := ref.Repository
	if strings.HasPrefix(hint, prefix+grammar.RepositorySeparator) {
		// try to keep hint identical, even across intermediate
		// artifact globalizations
		hint = hint[len(prefix)+1:]
	}
	if ref.Tag != nil {
		hint += grammar.TagSeparator + *ref.Tag
	}
	return hint
}

func (_ *AccessSpec) GetType() string {
	return Type
}

func (a *AccessSpec) AccessMethod(c cpi.ComponentVersionAccess) (cpi.AccessMethod, error) {
	return newMethod(c, a)
}

////////////////////////////////////////////////////////////////////////////////

type AccessMethod = *accessMethod

type accessMethod struct {
	lock sync.Mutex
	comp cpi.ComponentVersionAccess
	spec *AccessSpec

	finalizer Finalizer
	err       error
	art       oci.ArtifactAccess
	ref       *oci.RefSpec
	blob      artifactset.ArtifactBlob
}

var (
	_ cpi.AccessMethod      = (*accessMethod)(nil)
	_ accessio.DigestSource = (*accessMethod)(nil)
)

func newMethod(c cpi.ComponentVersionAccess, a *AccessSpec) (*accessMethod, error) {
	return &accessMethod{
		spec: a,
		comp: c,
	}, nil
}

func (m *accessMethod) GetKind() string {
	return Type
}

func (m *accessMethod) AccessSpec() cpi.AccessSpec {
	return m.spec
}

func (m *accessMethod) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.blob = nil
	return m.finalizer.Finalize()
}

func (m *accessMethod) eval() (oci.Repository, *oci.RefSpec, error) {
	ref, err := oci.ParseRef(m.spec.ImageReference)
	if err != nil {
		return nil, nil, err
	}
	ocictx := m.comp.GetContext().OCIContext()
	spec := ocictx.GetAlias(ref.Host)
	if spec == nil {
		spec = ocireg.NewRepositorySpec(ref.Host)
	}
	repo, err := ocictx.RepositoryForSpec(spec)
	return repo, &ref, err
}

func (m *accessMethod) GetArtifact(finalizer *Finalizer) (oci.ArtifactAccess, *oci.RefSpec, error) {
	repo, ref, err := m.eval()
	if err != nil {
		return nil, nil, err
	}
	finalizer.Close(repo)
	art, err := repo.LookupArtifact(ref.Repository, ref.Version())
	return art, ref, err
}

func (m *accessMethod) getArtifact() (oci.ArtifactAccess, *oci.RefSpec, error) {
	if m.art == nil && m.err == nil {
		m.art, m.ref, m.err = m.GetArtifact(&m.finalizer)
		if m.err == nil {
			m.finalizer.Close(m.art)
		}
	}
	return m.art, m.ref, m.err
}

func (m *accessMethod) Digest() digest.Digest {
	d, _ := m.GetDigest()
	return d
}

func (m *accessMethod) GetDigest() (digest.Digest, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	art, _, err := m.getArtifact()
	if err == nil {
		m.art = art
		blob, err := art.Blob()
		if err == nil {
			return blob.Digest(), nil
		}
		m.finalizer.Close(blob)
	}
	return "", err
}

func (m *accessMethod) Get() ([]byte, error) {
	blob, err := m.getBlob()
	if err != nil {
		return nil, err
	}
	return blob.Get()
}

func (m *accessMethod) Reader() (io.ReadCloser, error) {
	b, err := m.getBlob()
	if err != nil {
		return nil, err
	}
	r, err := b.Reader()
	if err != nil {
		return nil, err
	}
	// return accessio.AddCloser(r, b, "synthesized artifact"), nil
	return r, nil
}

func (m *accessMethod) MimeType() string {
	m.lock.Lock()
	defer m.lock.Unlock()

	art, _, err := m.getArtifact()
	if err != nil {
		return ""
	}
	return artdesc.ToContentMediaType(art.GetDescriptor().MimeType()) + artifactset.SynthesizedBlobFormat
}

func (m *accessMethod) getBlob() (artifactset.ArtifactBlob, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.blob != nil {
		return m.blob, nil
	}

	art, ref, err := m.getArtifact()
	if err != nil {
		return nil, err
	}
	logger := Logger(m.comp)
	logger.Info("synthesize artifact blob", "ref", m.spec.ImageReference)
	m.blob, err = artifactset.SynthesizeArtifactBlobForArtifact(art, ref.Version())
	logger.Info("synthesize artifact blob done", "ref", m.spec.ImageReference, "error", logging.ErrorMessage(err))
	if err != nil {
		return nil, err
	}
	m.finalizer.Close(m.blob)
	return m.blob, nil
}
