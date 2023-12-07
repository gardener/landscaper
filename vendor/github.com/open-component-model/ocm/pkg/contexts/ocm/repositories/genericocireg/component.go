// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/repocpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/utils"
)

const META_SEPARATOR = ".build-"

////////////////////////////////////////////////////////////////////////////////

type componentAccessImpl struct {
	bridge    repocpi.ComponentAccessBridge
	repo      *RepositoryImpl
	name      string
	namespace oci.NamespaceAccess
}

func newComponentAccess(repo *RepositoryImpl, name string, main bool) (*repocpi.ComponentAccessInfo, error) {
	mapped, err := repo.MapComponentNameToNamespace(name)
	if err != nil {
		return nil, err
	}
	namespace, err := repo.ocirepo.LookupNamespace(mapped)
	if err != nil {
		return nil, err
	}
	impl := &componentAccessImpl{
		repo:      repo,
		name:      name,
		namespace: namespace,
	}
	return &repocpi.ComponentAccessInfo{impl, "OCM component[OCI]", main}, nil
}

func (c *componentAccessImpl) Close() error {
	refmgmt.AllocLog.Trace("closing component [OCI]", "name", c.name)
	err := c.namespace.Close()
	refmgmt.AllocLog.Trace("closed component [OCI]", "name", c.name)
	return err
}

func (c *componentAccessImpl) SetBridge(base repocpi.ComponentAccessBridge) {
	c.bridge = base
}

func (c *componentAccessImpl) GetParentBridge() repocpi.RepositoryViewManager {
	return c.repo.bridge
}

func (c *componentAccessImpl) GetContext() cpi.Context {
	return c.repo.GetContext()
}

func (c *componentAccessImpl) GetName() string {
	return c.name
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

func (c *componentAccessImpl) LookupVersion(version string) (*repocpi.ComponentVersionAccessInfo, error) {
	acc, err := c.namespace.GetArtifact(toTag(version))
	if err != nil {
		if errors.IsErrNotFound(err) {
			return nil, cpi.ErrComponentVersionNotFoundWrap(err, c.name, version)
		}
		return nil, err
	}
	return newComponentVersionAccess(accessobj.ACC_WRITABLE, c, version, acc, true)
}

func (c *componentAccessImpl) NewVersion(version string, overrides ...bool) (*repocpi.ComponentVersionAccessInfo, error) {
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
