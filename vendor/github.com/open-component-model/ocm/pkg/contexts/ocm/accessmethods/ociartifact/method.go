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

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	ociidentity "github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	ocmcpi "github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
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
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](Type, accspeccpi.WithDescription(usage)))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](TypeV1, accspeccpi.WithFormatSpec(formatV1), accspeccpi.WithConfigHandler(ConfigHandler())))

	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](LegacyType))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](LegacyTypeV1))
}

func Is(spec accspeccpi.AccessSpec) bool {
	return spec != nil && (spec.GetKind() == Type || spec.GetKind() == LegacyType)
}

// AccessSpec describes the access for a oci registry.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`

	// ImageReference is the actual reference to the oci image repository and tag.
	ImageReference string `json:"imageReference"`
}

var (
	_ accspeccpi.AccessSpec   = (*AccessSpec)(nil)
	_ accspeccpi.HintProvider = (*AccessSpec)(nil)
	_ blobaccess.DigestSource = (*AccessSpec)(nil)
)

// New creates a new oci registry access spec version v1.
func New(ref string) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		ImageReference:      ref,
	}
}

func (a *AccessSpec) Describe(ctx accspeccpi.Context) string {
	return fmt.Sprintf("OCI artifact %s", a.ImageReference)
}

func (_ *AccessSpec) IsLocal(accspeccpi.Context) bool {
	return false
}

func (a *AccessSpec) Digest() digest.Digest {
	ref, err := oci.ParseRef(a.ImageReference)
	if err != nil || ref.Digest == nil {
		return ""
	}
	return *ref.Digest
}

func (a *AccessSpec) GlobalAccessSpec(ctx accspeccpi.Context) accspeccpi.AccessSpec {
	return a
}

func (a *AccessSpec) GetReferenceHint(cv accspeccpi.ComponentVersionAccess) string {
	ref, err := oci.ParseRef(a.ImageReference)
	if err != nil {
		return ""
	}
	hint := ref.Repository
	r := cv.Repository()
	if r != nil {
		prefix := ocmcpi.RepositoryPrefix(cv.Repository().GetSpecification())
		if strings.HasPrefix(hint, prefix+grammar.RepositorySeparator) {
			// try to keep hint identical, even across intermediate
			// artifact globalizations
			hint = hint[len(prefix)+1:]
		}
	}
	if ref.Tag != nil {
		hint += grammar.TagSeparator + *ref.Tag
	}
	return hint
}

func (a *AccessSpec) GetOCIReference(cv accspeccpi.ComponentVersionAccess) (string, error) {
	return a.ImageReference, nil
}

func (_ *AccessSpec) GetType() string {
	return Type
}

func (a *AccessSpec) AccessMethod(c accspeccpi.ComponentVersionAccess) (accspeccpi.AccessMethod, error) {
	return NewMethod(c.GetContext(), a, a.ImageReference)
}

func (a *AccessSpec) GetInexpensiveContentVersionIdentity(cv accspeccpi.ComponentVersionAccess) string {
	ref, err := oci.ParseRef(a.ImageReference)
	if err != nil {
		return ""
	}
	if ref.Digest != nil {
		return ref.Digest.String()
	}
	// TODO: optimize for oci registries
	return ""
}

////////////////////////////////////////////////////////////////////////////////

type AccessMethodImpl = *accessMethod

type accessMethod struct {
	lock      sync.Mutex
	ctx       accspeccpi.Context
	spec      accspeccpi.AccessSpec
	reference string

	finalizer Finalizer
	err       error

	id     credentials.ConsumerIdentity
	ref    *oci.RefSpec
	mime   string
	digest digest.Digest
	art    oci.ArtifactAccess

	repo oci.Repository
	blob artifactset.ArtifactBlob
}

var (
	_ accspeccpi.AccessMethodImpl          = (*accessMethod)(nil)
	_ blobaccess.DigestSource              = (*accessMethod)(nil)
	_ accspeccpi.DigestSource              = (*accessMethod)(nil)
	_ credentials.ConsumerIdentityProvider = (*accessMethod)(nil)
)

func NewMethod(ctx accspeccpi.ContextProvider, a accspeccpi.AccessSpec, ref string, repo ...oci.Repository) (accspeccpi.AccessMethod, error) {
	m := &accessMethod{
		spec:      a,
		reference: ref,
		ctx:       ctx.OCMContext(),
	}
	return accspeccpi.AccessMethodForImplementation(m, m.eval(utils.Optional(repo...)))
}

func (_ *accessMethod) IsLocal() bool {
	return false
}

func (m *accessMethod) GetOCIReference(cv accspeccpi.ComponentVersionAccess) (string, error) {
	return m.reference, nil
}

func (m *accessMethod) GetKind() string {
	return m.spec.GetKind()
}

func (m *accessMethod) AccessSpec() accspeccpi.AccessSpec {
	return m.spec
}

func (m *accessMethod) Cache() {
	m.lock.Lock()
	ref := m.ref
	m.lock.Unlock()
	if ref == nil {
		return
	}
	logger := Logger(WrapContextProvider(m.ctx))
	logger.Info("cache artifact blob", "ref", m.reference)

	_, m.err = m.getBlob()

	m.finalizer.Finalize()
}

func (m *accessMethod) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	list := errors.ErrorList{}

	if m.blob != nil {
		list.Add(m.blob.Close())
	}
	m.blob = nil
	m.art = nil
	m.ref = nil
	list.Add(m.finalizer.Finalize())
	return list.Result()
}

func (m *accessMethod) eval(relto oci.Repository) error {
	var (
		err error
		ref oci.RefSpec
	)

	if relto == nil {
		ref, err = oci.ParseRef(m.reference)
		if err != nil {
			return err
		}
		ocictx := m.ctx.OCIContext()
		spec := ocictx.GetAlias(ref.Host)
		if spec == nil {
			spec = ocireg.NewRepositorySpec(ref.Host)
		}
		repo, err := ocictx.RepositoryForSpec(spec)
		if err != nil {
			return err
		}
		m.finalizer.Close(repo, "repository for accessing %s", m.reference)
		m.repo = repo
	} else {
		repo, err := relto.Dup()
		if err != nil {
			return err
		}
		m.finalizer.Close(repo)
		art, err := oci.ParseArt(m.reference)
		if err != nil {
			return err
		}
		ref = oci.RefSpec{
			UniformRepositorySpec: *repo.GetSpecification().UniformRepositorySpec(),
			ArtSpec:               art,
		}
		m.repo = repo
	}

	m.ref = &ref
	m.id = credentials.GetProvidedConsumerId(m.repo, credentials.StringUsageContext(ref.Repository))
	return nil
}

func (m *accessMethod) GetArtifact() (oci.ArtifactAccess, *oci.RefSpec, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	err := m.getArtifact()
	if err != nil {
		return nil, nil, m.err
	}
	art := m.art
	if art != nil {
		art, err = art.Dup()
		if err != nil {
			return nil, nil, err
		}
	}
	return art, m.ref, err
}

func (m *accessMethod) getArtifact() error {
	if m.art == nil && m.err == nil && m.ref != nil {
		art, err := m.repo.LookupArtifact(m.ref.Repository, m.ref.Version())
		m.finalizer.Close(art, "artifact for accessing %s", m.reference)
		m.art, m.err = art, err
		m.mime = artdesc.ToContentMediaType(m.art.GetDescriptor().MimeType()) + artifactset.SynthesizedBlobFormat
		m.digest = art.Digest()
	}
	return m.err
}

func (m *accessMethod) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.id
}

func (m *accessMethod) GetIdentityMatcher() string {
	return ociidentity.CONSUMER_TYPE
}

func (m *accessMethod) Digest() digest.Digest {
	d, _ := m.GetDigest()
	return d
}

func (m *accessMethod) GetDigest() (digest.Digest, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	err := m.getArtifact()
	return m.digest, err
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
	return r, nil
}

func (m *accessMethod) MimeType() string {
	if m.mime == "" {
		m.lock.Lock()
		defer m.lock.Unlock()
		m.getArtifact()
	}
	return m.mime
}

func (m *accessMethod) getBlob() (artifactset.ArtifactBlob, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.blob != nil || m.err != nil {
		return m.blob, m.err
	}

	err := m.getArtifact()
	if err != nil {
		return nil, err
	}
	logger := Logger(WrapContextProvider(m.ctx))
	logger.Info("synthesize artifact blob", "ref", m.reference)
	m.blob, err = artifactset.SynthesizeArtifactBlobForArtifact(m.art, m.ref.Version())
	logger.Info("synthesize artifact blob done", "ref", m.reference, "error", logging.ErrorMessage(err))
	if err != nil {
		m.err = err
		return nil, err
	}
	return m.blob, nil
}
