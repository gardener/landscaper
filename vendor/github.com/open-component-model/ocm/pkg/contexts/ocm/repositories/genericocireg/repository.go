// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	ocicpi "github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg/componentmapping"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type OCIBasedRepository interface {
	OCIRepository() ocicpi.Repository
}

func GetOCIRepository(r cpi.Repository) ocicpi.Repository {
	if o, ok := r.(OCIBasedRepository); ok {
		return o.OCIRepository()
	}
	impl, err := cpi.GetRepositoryImplementation(r)
	if err != nil {
		return nil
	}
	if o, ok := impl.(OCIBasedRepository); ok {
		return o.OCIRepository()
	}
	return nil
}

type _RepositoryImplBase = cpi.RepositoryImplBase

type RepositoryImpl struct {
	_RepositoryImplBase
	meta    ComponentRepositoryMeta
	nonref  cpi.Repository
	ocirepo oci.Repository
}

var (
	_ cpi.RepositoryImpl                   = (*RepositoryImpl)(nil)
	_ credentials.ConsumerIdentityProvider = (*RepositoryImpl)(nil)
)

func NewRepository(ctx cpi.Context, meta *ComponentRepositoryMeta, ocirepo oci.Repository) (cpi.Repository, error) {
	impl := &RepositoryImpl{
		_RepositoryImplBase: *cpi.NewRepositoryImplBase(ctx.OCMContext()),
		meta:                *DefaultComponentRepositoryMeta(meta),
		ocirepo:             ocirepo,
	}
	impl.nonref = cpi.NewNoneRefRepositoryView(impl)
	r := cpi.NewRepository(impl, "OCM repo[OCI]")
	return r, nil
}

func (r *RepositoryImpl) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	prefix := r.meta.SubPath
	if c, ok := utils.Optional(uctx...).(credentials.StringUsageContext); ok {
		prefix = path.Join(prefix, componentmapping.ComponentDescriptorNamespace, c.String())
	}
	if p, ok := r.ocirepo.(credentials.ConsumerIdentityProvider); ok {
		return p.GetConsumerId(credentials.StringUsageContext(prefix))
	}
	return nil
}

func (r *RepositoryImpl) GetIdentityMatcher() string {
	if p, ok := r.ocirepo.(credentials.ConsumerIdentityProvider); ok {
		return p.GetIdentityMatcher()
	}
	return ""
}

func (r *RepositoryImpl) Close() error {
	return r.ocirepo.Close()
}

func (r *RepositoryImpl) OCIRepository() ocicpi.Repository {
	return r.ocirepo
}

func (r *RepositoryImpl) GetSpecification() cpi.RepositorySpec {
	return &RepositorySpec{
		RepositorySpec:          r.ocirepo.GetSpecification(),
		ComponentRepositoryMeta: r.meta,
	}
}

func (r *RepositoryImpl) ComponentLister() cpi.ComponentLister {
	if r.meta.ComponentNameMapping != OCIRegistryURLPathMapping {
		return nil
	}
	lister := r.ocirepo.NamespaceLister()
	if lister == nil {
		return nil
	}
	return r
}

func (r *RepositoryImpl) NumComponents(prefix string) (int, error) {
	lister := r.ocirepo.NamespaceLister()
	if lister == nil {
		return -1, errors.ErrNotSupported("component lister")
	}
	p := path.Join(r.meta.SubPath, componentmapping.ComponentDescriptorNamespace, prefix)
	if strings.HasSuffix(prefix, "/") && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return lister.NumNamespaces(p)
}

func (r *RepositoryImpl) GetComponents(prefix string, closure bool) ([]string, error) {
	lister := r.ocirepo.NamespaceLister()
	if lister == nil {
		return nil, errors.ErrNotSupported("component lister")
	}
	p := path.Join(r.meta.SubPath, componentmapping.ComponentDescriptorNamespace)
	compprefix := len(p) + 1
	p = path.Join(p, prefix)
	if strings.HasSuffix(prefix, "/") && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	tmp, err := lister.GetNamespaces(p, closure)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(tmp))
	for i, r := range tmp {
		result[i] = r[compprefix:]
	}
	return result, nil
}

func (r *RepositoryImpl) GetOCIRepository() oci.Repository {
	return r.ocirepo
}

func (r *RepositoryImpl) ExistsComponentVersion(name string, version string) (bool, error) {
	namespace, err := r.MapComponentNameToNamespace(name)
	if err != nil {
		return false, err
	}
	a, err := r.ocirepo.LookupArtifact(namespace, version)
	if err != nil {
		return false, err
	}
	defer a.Close()
	desc, err := a.Manifest()
	if err != nil {
		return false, err
	}
	switch desc.Config.MediaType {
	case componentmapping.ComponentDescriptorConfigMimeType,
		componentmapping.LegacyComponentDescriptorConfigMimeType,
		componentmapping.Legacy2ComponentDescriptorConfigMimeType:
		return true, nil
	}
	return false, nil
}

func (r *RepositoryImpl) LookupComponent(name string) (cpi.ComponentAccess, error) {
	return newComponentAccess(r, name, true)
}

func (r *RepositoryImpl) LookupComponentVersion(name string, version string) (cpi.ComponentVersionAccess, error) {
	c, err := newComponentAccess(r, name, false)
	if err != nil {
		return nil, err
	}
	defer accessio.PropagateCloseTemporary(&err, c) // temporary component object not exposed.
	return c.LookupVersion(version)
}

func (r *RepositoryImpl) MapComponentNameToNamespace(name string) (string, error) {
	switch r.meta.ComponentNameMapping {
	case OCIRegistryURLPathMapping, "":
		return path.Join(r.meta.SubPath, componentmapping.ComponentDescriptorNamespace, name), nil
	case OCIRegistryDigestMapping:
		h := sha256.New()
		_, _ = h.Write([]byte(name))
		return path.Join(r.meta.SubPath, hex.EncodeToString(h.Sum(nil))), nil
	default:
		return "", fmt.Errorf("unknown component name mapping method %s", r.meta.ComponentNameMapping)
	}
}
