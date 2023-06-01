// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"fmt"
	"path"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localociblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	ocihdlr "github.com/open-component-model/ocm/pkg/contexts/ocm/blobhandler/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/support"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
)

type ComponentVersion struct {
	container *ComponentVersionContainer
	*support.ComponentVersionAccess
}

var _ cpi.ComponentVersionAccess = (*ComponentVersion)(nil)

// newComponentVersionAccess creates an component access for the artifact access, if this fails the artifact acess is closed.
func newComponentVersionAccess(mode accessobj.AccessMode, comp *componentAccessImpl, version string, access oci.ArtifactAccess, persistent bool) (*ComponentVersion, error) {
	c, err := newComponentVersionContainer(mode, comp, version, access)
	if err != nil {
		return nil, err
	}
	acc, err := support.NewComponentVersionAccess(c, true, persistent)
	if err != nil {
		c.Close()
		return nil, err
	}
	return &ComponentVersion{
		container:              c,
		ComponentVersionAccess: acc,
	}, nil
}

// //////////////////////////////////////////////////////////////////////////////

type ComponentVersionContainer struct {
	comp     *ComponentAccess
	version  string
	access   oci.ArtifactAccess
	manifest oci.ManifestAccess
	state    accessobj.State
}

var _ support.ComponentVersionContainer = (*ComponentVersionContainer)(nil)

func newComponentVersionContainer(mode accessobj.AccessMode, comp *componentAccessImpl, version string, access oci.ArtifactAccess) (*ComponentVersionContainer, error) {
	m := access.ManifestAccess()
	if m == nil {
		return nil, errors.ErrInvalid("artifact type")
	}
	state, err := NewState(mode, comp.name, version, m)
	if err != nil {
		access.Close()
		return nil, err
	}
	v, err := comp.View(false)
	if err != nil {
		access.Close()
		return nil, err
	}
	return &ComponentVersionContainer{
		comp:     v,
		version:  version,
		access:   access,
		manifest: m,
		state:    state,
	}, nil
}

func (c *ComponentVersionContainer) Repository() cpi.Repository {
	return c.comp.repo
}

func (c *ComponentVersionContainer) Close() error {
	if c.manifest == nil {
		return accessio.ErrClosed
	}

	c.manifest = nil

	err := c.access.Close()
	if err != nil {
		c.comp.Close()

		return fmt.Errorf("failed to close access artifact access: %w", err)
	}

	return c.comp.Close()
}

func (c *ComponentVersionContainer) Check() error {
	if c.version != c.GetDescriptor().Version {
		return errors.ErrInvalid("component version", c.GetDescriptor().Version)
	}
	if c.comp.name != c.GetDescriptor().Name {
		return errors.ErrInvalid("component name", c.GetDescriptor().Name)
	}
	return nil
}

func (c *ComponentVersionContainer) GetContext() cpi.Context {
	return c.comp.GetContext()
}

func (c *ComponentVersionContainer) ComponentAccess() cpi.ComponentAccess {
	return c.comp
}

func (c *ComponentVersionContainer) IsReadOnly() bool {
	return c.state.IsReadOnly()
}

func (c *ComponentVersionContainer) IsClosed() bool {
	return c.manifest == nil
}

func (c *ComponentVersionContainer) AccessMethod(a cpi.AccessSpec) (cpi.AccessMethod, error) {
	if a.GetKind() == localblob.Type {
		accessSpec, err := c.comp.GetContext().AccessSpecForSpec(a)
		if err != nil {
			return nil, err
		}
		return newLocalBlobAccessMethod(accessSpec.(*localblob.AccessSpec), c.comp.namespace, c.access), nil
	}
	if a.GetKind() == localociblob.Type {
		accessSpec, err := c.comp.GetContext().AccessSpecForSpec(a)
		if err != nil {
			return nil, err
		}
		return newLocalOCIBlobAccessMethod(accessSpec.(*localblob.AccessSpec), c.comp.namespace, c.access), nil
	}
	return nil, errors.ErrNotSupported(errors.KIND_ACCESSMETHOD, a.GetType(), "oci registry")
}

func (c *ComponentVersionContainer) Update() error {
	logger := Logger(c.GetContext()).WithValues("cv", common.NewNameVersion(c.comp.name, c.version))
	err := c.Check()
	if err != nil {
		return fmt.Errorf("check failed: %w", err)
	}

	if c.state.HasChanged() {
		logger.Debug("update component version")
		desc := c.GetDescriptor()
		layers := generics.Set[int]{}
		for i := range c.manifest.GetDescriptor().Layers {
			layers.Add(i)
		}
		for i, r := range desc.Resources {
			s, l, err := c.evalLayer(r.Access)
			if err != nil {
				return fmt.Errorf("failed resource layer evaluation: %w", err)
			}
			if l > 0 {
				layers.Delete(l)
			}
			if s != r.Access {
				desc.Resources[i].Access = s
			}
		}
		for i, r := range desc.Sources {
			s, l, err := c.evalLayer(r.Access)
			if err != nil {
				return fmt.Errorf("failed source layer evaluation: %w", err)
			}
			if l > 0 {
				layers.Delete(l)
			}
			if s != r.Access {
				desc.Sources[i].Access = s
			}
		}
		m := c.manifest.GetDescriptor()
		i := len(m.Layers) - 1

		for i > 0 {
			if layers.Contains(i) {
				logger.Debug("removing unused layer", "layer", i)
				m.Layers = append(m.Layers[:i], m.Layers[i+1:]...)
			}
			i--
		}
		if _, err := c.state.Update(); err != nil {
			return fmt.Errorf("failed to update state: %w", err)
		}

		logger.Debug("add oci artifact")
		if _, err := c.comp.namespace.AddArtifact(c.manifest, toTag(c.version)); err != nil {
			return fmt.Errorf("unable to add artifact: %w", err)
		}
	}

	return nil
}

func (c *ComponentVersionContainer) evalLayer(s compdesc.AccessSpec) (compdesc.AccessSpec, int, error) {
	var d *artdesc.Descriptor

	spec, err := c.GetContext().AccessSpecForSpec(s)
	if err != nil {
		return s, 0, err
	}
	if a, ok := spec.(*localblob.AccessSpec); ok {
		if ok, _ := artdesc.IsDigest(a.LocalReference); !ok {
			return s, 0, errors.ErrInvalid("digest", a.LocalReference)
		}
		d = &artdesc.Descriptor{Digest: digest.Digest(a.LocalReference), MediaType: a.GetMimeType()}
	}
	if a, ok := spec.(*localblob.AccessSpec); ok {
		if ok, _ := artdesc.IsDigest(a.LocalReference); !ok {
			return s, 0, errors.ErrInvalid("digest", a.LocalReference)
		}
		d = &artdesc.Descriptor{Digest: digest.Digest(a.LocalReference), MediaType: a.GetMimeType()}
	}
	if d != nil {
		// find layer
		layers := c.manifest.GetDescriptor().Layers
		max := len(layers) - 1
		for i := range layers {
			l := layers[len(layers)-1-i]
			if i < max && l.Digest == d.Digest && (d.Digest == "" || d.Digest == l.Digest) {
				return s, len(layers) - 1 - i, nil
			}
		}
		return s, 0, fmt.Errorf("resource access %s: no layer found for local blob %s[%s]", spec.Describe(c.GetContext()), d.Digest, d.MediaType)
	}
	return s, 0, nil
}

func (c *ComponentVersionContainer) GetDescriptor() *compdesc.ComponentDescriptor {
	return c.state.GetState().(*compdesc.ComponentDescriptor)
}

func (c *ComponentVersionContainer) GetBlobData(name string) (cpi.DataAccess, error) {
	return c.manifest.GetBlob(digest.Digest((name)))
}

func (c *ComponentVersionContainer) GetStorageContext(cv cpi.ComponentVersionAccess) cpi.StorageContext {
	return ocihdlr.New(c.comp.repo, cv, c.comp.repo.ocirepo.GetSpecification().GetKind(), c.comp.repo.ocirepo, c.comp.namespace, c.manifest)
}

func (c *ComponentVersionContainer) AddBlobFor(storagectx cpi.StorageContext, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	if blob == nil {
		return nil, errors.New("a resource has to be defined")
	}

	err := c.manifest.AddBlob(blob)
	if err != nil {
		return nil, err
	}
	err = storagectx.(*ocihdlr.StorageContext).AssureLayer(blob)
	if err != nil {
		return nil, err
	}
	return localblob.New(blob.Digest().String(), refName, blob.MimeType(), global), nil
}

// AssureGlobalRef provides a global manifest for a local OCI Artifact.
func (c *ComponentVersionContainer) AssureGlobalRef(d digest.Digest, url, name string) (cpi.AccessSpec, error) {
	blob, err := c.manifest.GetBlob(d)
	if err != nil {
		return nil, err
	}
	var namespace oci.NamespaceAccess
	var version string
	var tag string
	if name == "" {
		namespace = c.comp.namespace
	} else {
		i := strings.LastIndex(name, ":")
		if i > 0 {
			version = name[i+1:]
			name = name[:i]
			tag = version
		}
		namespace, err = c.comp.repo.ocirepo.LookupNamespace(name)
		if err != nil {
			return nil, err
		}
	}
	set, err := artifactset.OpenFromBlob(accessobj.ACC_READONLY, blob)
	if err != nil {
		return nil, err
	}
	defer set.Close()
	digest := set.GetMain()
	if version == "" {
		version = digest.String()
	}
	art, err := set.GetArtifact(digest.String())
	if err != nil {
		return nil, err
	}
	err = artifactset.TransferArtifact(art, namespace, oci.AsTags(tag)...)
	if err != nil {
		return nil, err
	}

	ref := path.Join(url+namespace.GetNamespace()) + ":" + version

	global := ociartifact.New(ref)
	return global, nil
}
