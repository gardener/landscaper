// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

const META_SEPARATOR = ".build-"

type ComponentAccess struct {
	view accessio.CloserView // handle close and refs
	*componentAccessImpl
}

// implemented by view
// the rest is directly taken from the artifact set implementation

func (s *ComponentAccess) Dup() (cpi.ComponentAccess, error) {
	if s.view.IsClosed() {
		return nil, accessio.ErrClosed
	}
	return s.componentAccessImpl.Dup()
}

func (s *ComponentAccess) Close() error {
	return s.view.Close()
}

func (s *ComponentAccess) IsClosed() bool {
	return s.view.IsClosed()
}

////////////////////////////////////////////////////////////////////////////////

type componentAccessImpl struct {
	refs      accessio.ReferencableCloser
	repo      *Repository
	name      string
	namespace oci.NamespaceAccess
}

var _ cpi.ComponentAccess = (*ComponentAccess)(nil)

func newComponentAccess(repo *RepositoryImpl, name string, main bool) (*ComponentAccess, error) {
	mapped, err := repo.MapComponentNameToNamespace(name)
	if err != nil {
		return nil, err
	}
	v, err := repo.View(false)
	if err != nil {
		return nil, err
	}
	namespace, err := repo.ocirepo.LookupNamespace(mapped)
	if err != nil {
		v.Close()
		return nil, err
	}
	n := &componentAccessImpl{
		repo:      v,
		name:      name,
		namespace: namespace,
	}
	n.refs = accessio.NewRefCloser(n, true)
	return n.View(main)
}

func (a *componentAccessImpl) Dup() (cpi.ComponentAccess, error) {
	return a.View(false)
}

func (a *componentAccessImpl) View(main ...bool) (*ComponentAccess, error) {
	v, err := a.refs.View(main...)
	if err != nil {
		return nil, err
	}
	return &ComponentAccess{view: v, componentAccessImpl: a}, nil
}

func (c *componentAccessImpl) GetName() string {
	return c.name
}

func (c *componentAccessImpl) Close() error {
	err := c.namespace.Close()
	if err != nil {
		c.repo.Close()
		return err
	}
	return c.repo.Close()
}

func (c *componentAccessImpl) GetContext() cpi.Context {
	return c.repo.GetContext()
}

////////////////////////////////////////////////////////////////////////////////

func toTag(v string) string {
	_, err := semver.NewVersion(v)
	if err != nil {
		panic(errors.Wrapf(err, "%s is no semver version", v))
	}
	return strings.ReplaceAll(v, "+", META_SEPARATOR)
}

func toVersion(t string) string {
	next := 0
	for {
		if idx := strings.Index(t[next:], META_SEPARATOR); idx >= 0 {
			next += idx + len(META_SEPARATOR)
		} else {
			break
		}
	}
	if next == 0 {
		return t
	}
	return t[:next-len(META_SEPARATOR)] + "+" + t[next:]
}

func (c *componentAccessImpl) ListVersions() ([]string, error) {
	tags, err := c.namespace.ListTags()
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		// omit reported digests (typically for ctf)
		if ok, _ := artdesc.IsDigest(t); !ok {
			result = append(result, toVersion(t))
		}
	}
	return result, err
}

func (c *componentAccessImpl) LookupVersion(version string) (cpi.ComponentVersionAccess, error) {
	v, err := c.View(false)
	if err != nil {
		return nil, err
	}
	defer v.Close()
	acc, err := c.namespace.GetArtifact(toTag(version))
	if err != nil {
		if errors.IsErrNotFound(err) {
			return nil, cpi.ErrComponentVersionNotFoundWrap(err, c.name, version)
		}
		return nil, err
	}
	return newComponentVersionAccess(accessobj.ACC_WRITABLE, c, version, acc, true)
}

func (c *componentAccessImpl) AddVersion(access cpi.ComponentVersionAccess) error {
	if a, ok := access.(*ComponentVersion); ok {
		if a.GetName() != c.GetName() {
			return errors.ErrInvalid("component name", a.GetName())
		}
		if a.container.comp.componentAccessImpl != c {
			return fmt.Errorf("cannot add component version: component version access %s not created for target", a.GetName()+":"+a.GetVersion())
		}
		a.EnablePersistence()
		return a.container.Update()
	}
	return errors.ErrInvalid("component version")
}

func (c *componentAccessImpl) NewVersion(version string, overrides ...bool) (cpi.ComponentVersionAccess, error) {
	v, err := c.View(false)
	if err != nil {
		return nil, err
	}
	defer v.Close()

	override := utils.Optional(overrides...)
	acc, err := c.namespace.GetArtifact(toTag(version))
	if err == nil {
		if override {
			return newComponentVersionAccess(accessobj.ACC_CREATE, c, version, acc, false)
		}
		return nil, errors.ErrAlreadyExists(cpi.KIND_COMPONENTVERSION, c.name+"/"+version)
	}
	if !errors.IsErrNotFoundKind(err, oci.KIND_OCIARTIFACT) {
		return nil, err
	}
	acc, err = c.namespace.NewArtifact()
	if err != nil {
		return nil, err
	}
	return newComponentVersionAccess(accessobj.ACC_CREATE, c, version, acc, false)
}
