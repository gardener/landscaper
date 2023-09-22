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
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/support"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

const META_SEPARATOR = ".build-"

////////////////////////////////////////////////////////////////////////////////

type _ComponentAccessImplBase = cpi.ComponentAccessImplBase

type componentAccessImpl struct {
	_ComponentAccessImplBase
	repo      *RepositoryImpl
	name      string
	namespace oci.NamespaceAccess
}

func newComponentAccess(repo *RepositoryImpl, name string, main bool) (cpi.ComponentAccess, error) {
	mapped, err := repo.MapComponentNameToNamespace(name)
	if err != nil {
		return nil, err
	}

	base, err := cpi.NewComponentAccessImplBase(repo.GetContext(), name, repo)
	if err != nil {
		return nil, err
	}
	namespace, err := repo.ocirepo.LookupNamespace(mapped)
	if err != nil {
		base.Close()
		return nil, err
	}
	impl := &componentAccessImpl{
		_ComponentAccessImplBase: *base,
		repo:                     repo,
		name:                     name,
		namespace:                namespace,
	}
	return cpi.NewComponentAccess(impl, "OCM component[OCI]"), nil
}

func (c *componentAccessImpl) Close() error {
	return accessio.Close(c.namespace, c._ComponentAccessImplBase)
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

func (c *componentAccessImpl) IsReadOnly() bool {
	// TODO: extend OCI to query ReadOnly mode
	return false
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

func (c *componentAccessImpl) HasVersion(vers string) (bool, error) {
	tags, err := c.namespace.ListTags()
	if err != nil {
		return false, err
	}
	for _, t := range tags {
		// omit reported digests (typically for ctf)
		if ok, _ := artdesc.IsDigest(t); !ok {
			if vers == t {
				return true, nil
			}
		}
	}
	return false, err
}

func (c *componentAccessImpl) LookupVersion(version string) (cpi.ComponentVersionAccess, error) {
	v, err := c.repo.View()
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
	if access.GetName() != c.GetName() {
		return errors.ErrInvalid("component name", access.GetName())
	}
	cont, err := support.GetComponentVersionContainer(access)
	if err != nil {
		return fmt.Errorf("cannot add component version: component version access %s not created for target", access.GetName()+":"+access.GetVersion())
	}
	mine, ok := cont.(*ComponentVersionContainer)
	if !ok || mine.comp != c {
		return fmt.Errorf("cannot add component version: component version access %s not created for target", access.GetName()+":"+access.GetVersion())
	}
	ok = mine.impl.EnablePersistence()
	if !ok {
		return fmt.Errorf("version has been discarded")
	}
	return mine.impl.Update(false)
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
