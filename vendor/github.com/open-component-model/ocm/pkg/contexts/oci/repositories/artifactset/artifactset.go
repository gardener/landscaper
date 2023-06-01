// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactset

import (
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi/support"
	"github.com/open-component-model/ocm/pkg/errors"
)

const (
	MAINARTIFACT_ANNOTATION = "software.ocm/main"
	TAGS_ANNOTATION         = "software.ocm/tags"
	TYPE_ANNOTATION         = "software.ocm/type"

	OCITAG_ANNOTATION = "org.opencontainers.image.ref.name"
)

func RetrieveMainArtifact(m map[string]string) string {
	return m[MAINARTIFACT_ANNOTATION]
}

func RetrieveTags(m map[string]string) string {
	return m[TAGS_ANNOTATION]
}

func RetrieveType(m map[string]string) string {
	return m[TYPE_ANNOTATION]
}

// ArtifactSet provides an artifact set view on the artifact set implementation.
// Every ArtifactSet is separated closable. If the last view is closed
// the implementation is released.
type ArtifactSet struct {
	*artifactSetImpl // provide the artifact set interface
}

// implemented by view
// the rest is directly taken from the artifact set implementation

func (s *ArtifactSet) Close() error {
	return s.view.Close()
}

func (s *ArtifactSet) IsClosed() bool {
	return s.view.IsClosed()
}

////////////////////////////////////////////////////////////////////////////////

type artifactSetImpl struct {
	view support.ArtifactSetContainer
	impl support.ArtifactSetContainerImpl
	base *FileSystemBlobAccess
	*support.ArtifactSetAccess
}

var (
	_ cpi.ArtifactSink    = (*ArtifactSet)(nil)
	_ cpi.NamespaceAccess = (*ArtifactSet)(nil)
)

// New returns a new representation based element.
func New(acc accessobj.AccessMode, fs vfs.FileSystem, setup accessobj.Setup, closer accessobj.Closer, mode vfs.FileMode, formatVersion string) (*ArtifactSet, error) {
	return _Wrap(accessobj.NewAccessObject(NewAccessObjectInfo(formatVersion), acc, fs, setup, closer, mode))
}

func _Wrap(obj *accessobj.AccessObject, err error) (*ArtifactSet, error) {
	if err != nil {
		return nil, err
	}
	s := &artifactSetImpl{
		base: NewFileSystemBlobAccess(obj),
	}
	s.ArtifactSetAccess = support.NewArtifactSetAccess(s)
	s.view, s.impl = support.NewArtifactSetContainer(s)
	return &ArtifactSet{s}, nil
}

func (a *artifactSetImpl) GetNamespace() string {
	return ""
}

func (a *artifactSetImpl) Annotate(name string, value string) {
	a.base.Lock()
	defer a.base.Unlock()

	d := a.GetIndex()
	if d.Annotations == nil {
		d.Annotations = map[string]string{}
	}
	d.Annotations[name] = value
}

////////////////////////////////////////////////////////////////////////////////
// sink

func (a *artifactSetImpl) AddTags(digest digest.Digest, tags ...string) error {
	if a.IsClosed() {
		return accessio.ErrClosed
	}
	if len(tags) == 0 {
		return nil
	}

	a.base.Lock()
	defer a.base.Unlock()

	idx := a.GetIndex()
	for i, e := range idx.Manifests {
		if e.Digest == digest {
			if e.Annotations == nil {
				e.Annotations = map[string]string{}
				idx.Manifests[i].Annotations = e.Annotations
			}
			cur := RetrieveTags(e.Annotations)
			if cur != "" {
				cur = strings.Join(append([]string{cur}, tags...), ",")
			} else {
				cur = strings.Join(tags, ",")
			}
			e.Annotations[TAGS_ANNOTATION] = cur
			if a.base.FileSystemBlobAccess.Access().GetInfo().GetDescriptorFileName() == OCIArtifactSetDescriptorFileName {
				e.Annotations[OCITAG_ANNOTATION] = tags[0]
			}
			return nil
		}
	}
	return errors.ErrUnknown(cpi.KIND_OCIARTIFACT, digest.String())
}

////////////////////////////////////////////////////////////////////////////////
// forward

func (a *artifactSetImpl) IsReadOnly() bool {
	return a.base.IsReadOnly()
}

func (a *artifactSetImpl) Write(path string, mode vfs.FileMode, opts ...accessio.Option) error {
	return a.base.Write(path, mode, opts...)
}

func (a *artifactSetImpl) Update() error {
	return a.base.Update()
}

func (a *artifactSetImpl) Close() error {
	return a.base.Close()
}

func (a *artifactSetImpl) IsClosed() bool {
	return a.base.IsClosed()
}

// GetIndex returns the index of the included artifacts
// (image manifests and image indices)
// The manifst entries may describe dedicated tags
// to use for the dedicated artifact as annotation
// with the key TAGS_ANNOTATION.
func (a *artifactSetImpl) GetIndex() *artdesc.Index {
	if a.IsReadOnly() {
		return a.base.GetState().GetOriginalState().(*artdesc.Index)
	}
	return a.base.GetState().GetState().(*artdesc.Index)
}

// GetMain returns the digest of the main artifact
// described by this artifact set.
// There might be more, if the main artifact is an index.
func (a *artifactSetImpl) GetMain() digest.Digest {
	idx := a.GetIndex()
	if idx.Annotations == nil {
		return ""
	}
	return digest.Digest(RetrieveMainArtifact(idx.Annotations))
}

func (a *artifactSetImpl) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	return a.GetIndex().GetBlobDescriptor(digest)
}

func (a *artifactSetImpl) GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error) {
	return a.base.GetBlobData(digest)
}

func (a *artifactSetImpl) AddBlob(blob cpi.BlobAccess) error {
	if a.IsClosed() {
		return accessio.ErrClosed
	}
	if a.IsReadOnly() {
		return accessio.ErrReadOnly
	}
	if blob == nil {
		return nil
	}
	a.base.Lock()
	defer a.base.Unlock()
	return a.base.AddBlob(blob)
}

func (a *artifactSetImpl) ListTags() ([]string, error) {
	result := []string{}
	for _, a := range a.GetIndex().Manifests {
		if a.Annotations != nil {
			if tags := RetrieveTags(a.Annotations); tags != "" {
				result = append(result, strings.Split(tags, ",")...)
			}
		}
	}
	return result, nil
}

func (a *artifactSetImpl) GetTags(digest digest.Digest) ([]string, error) {
	result := []string{}
	for _, a := range a.GetIndex().Manifests {
		if a.Digest == digest && a.Annotations != nil {
			if tags := RetrieveTags(a.Annotations); tags != "" {
				result = append(result, strings.Split(tags, ",")...)
			}
		}
	}
	return result, nil
}

func (a *artifactSetImpl) HasArtifact(ref string) (bool, error) {
	if a.IsClosed() {
		return false, accessio.ErrClosed
	}
	a.base.Lock()
	defer a.base.Unlock()
	return a.hasArtifact(ref)
}

func (a *artifactSetImpl) GetArtifact(ref string) (cpi.ArtifactAccess, error) {
	if a.IsClosed() {
		return nil, accessio.ErrClosed
	}
	a.base.Lock()
	defer a.base.Unlock()
	return a.getArtifact(ref)
}

func (a *artifactSetImpl) matcher(ref string) func(d *artdesc.Descriptor) bool {
	if ok, digest := artdesc.IsDigest(ref); ok {
		return func(desc *artdesc.Descriptor) bool {
			return desc.Digest == digest
		}
	}
	return func(d *artdesc.Descriptor) bool {
		if d.Annotations == nil {
			return false
		}
		for _, tag := range strings.Split(RetrieveTags(d.Annotations), ",") {
			if tag == ref {
				return true
			}
		}
		return false
	}
}

func (a *artifactSetImpl) hasArtifact(ref string) (bool, error) {
	idx := a.GetIndex()
	match := a.matcher(ref)
	for i := range idx.Manifests {
		if match(&idx.Manifests[i]) {
			return true, nil
		}
	}
	return false, nil
}

func (a *artifactSetImpl) getArtifact(ref string) (cpi.ArtifactAccess, error) {
	idx := a.GetIndex()
	match := a.matcher(ref)
	for i, e := range idx.Manifests {
		if match(&idx.Manifests[i]) {
			return a.base.GetArtifact(a.impl, e.Digest)
		}
	}
	return nil, errors.ErrUnknown(cpi.KIND_OCIARTIFACT, ref)
}

func (a *artifactSetImpl) AnnotateArtifact(digest digest.Digest, name, value string) error {
	if a.IsClosed() {
		return accessio.ErrClosed
	}
	if a.IsReadOnly() {
		return accessio.ErrReadOnly
	}
	a.base.Lock()
	defer a.base.Unlock()
	idx := a.GetIndex()
	for i, e := range idx.Manifests {
		if e.Digest == digest {
			annos := e.Annotations
			if annos == nil {
				annos = map[string]string{}
				idx.Manifests[i].Annotations = annos
			}
			annos[name] = value
			return nil
		}
	}
	return errors.ErrUnknown(cpi.KIND_OCIARTIFACT, digest.String())
}

func (a *artifactSetImpl) AddArtifact(artifact cpi.Artifact, tags ...string) (access accessio.BlobAccess, err error) {
	blob, err := a.AddPlatformArtifact(artifact, nil)
	if err != nil {
		return nil, err
	}
	return blob, a.AddTags(blob.Digest(), tags...)
}

func (a *artifactSetImpl) AddPlatformArtifact(artifact cpi.Artifact, platform *artdesc.Platform) (access accessio.BlobAccess, err error) {
	if a.IsClosed() {
		return nil, accessio.ErrClosed
	}
	if a.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	a.base.Lock()
	defer a.base.Unlock()
	idx := a.GetIndex()
	blob, err := a.base.AddArtifactBlob(artifact)
	if err != nil {
		return nil, err
	}

	idx.Manifests = append(idx.Manifests, cpi.Descriptor{
		MediaType:   blob.MimeType(),
		Digest:      blob.Digest(),
		Size:        blob.Size(),
		URLs:        nil,
		Annotations: nil,
		Platform:    platform,
	})
	return blob, nil
}

func (a *artifactSetImpl) NewArtifact(artifact ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
	if a.IsClosed() {
		return nil, accessio.ErrClosed
	}
	if a.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	return support.NewArtifact(a.impl, artifact...)
}
