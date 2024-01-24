// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package composition

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/virtual"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

////////////////////////////////////////////////////////////////////////////////

func NewRepository(ctx cpi.ContextProvider, names ...string) cpi.Repository {
	var repositories *Repositories

	name := utils.Optional(names...)
	if name != "" {
		repositories = ctx.OCMContext().GetAttributes().GetOrCreateAttribute(ATTR_REPOS, newRepositories).(*Repositories)
		if repo := repositories.GetRepository(name); repo != nil {
			repo, _ = repo.Dup()
			return repo
		}
	}
	repo := virtual.NewRepository(ctx.OCMContext(), NewAccess())
	if repositories != nil {
		repositories.SetRepository(name, repo)
		repo, _ = repo.Dup()
	}
	return repo
}

type Index = virtual.Index[common.NameVersion]

type Access struct {
	lock  sync.Mutex
	index *Index
	blobs map[string]blobaccess.BlobAccess
}

var _ virtual.Access = (*Access)(nil)

func NewAccess() *Access {
	return &Access{
		index: virtual.NewIndex[common.NameVersion](),
		blobs: map[string]blobaccess.BlobAccess{},
	}
}

func (a *Access) IsReadOnly() bool {
	return false
}

func (a *Access) ComponentLister() cpi.ComponentLister {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.index
}

func (a *Access) ExistsComponentVersion(name string, version string) (bool, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	e := a.index.Get(name, version)
	return e != nil, nil
}

func (a *Access) ListVersions(comp string) ([]string, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.index.GetVersions(comp), nil
}

func (a *Access) GetComponentVersion(comp, version string) (virtual.VersionAccess, error) {
	var cd *compdesc.ComponentDescriptor

	a.lock.Lock()
	defer a.lock.Unlock()

	i := a.index.Get(comp, version)
	if i == nil {
		cd = compdesc.New(comp, version)
		err := a.index.Add(cd, common.VersionedElementKey(cd))
		if err != nil {
			return nil, err
		}
	} else {
		cd = i.CD()
	}
	return &VersionAccess{a, cd.GetName(), cd.GetVersion(), cd.Copy()}, nil
}

func (a *Access) GetBlob(name string) (blobaccess.BlobAccess, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	b := a.blobs[name]
	if b == nil {
		return nil, errors.ErrNotFound(blobaccess.KIND_BLOB, name)
	}
	return b.Dup()
}

func (a *Access) AddBlob(blob blobaccess.BlobAccess) (string, error) {
	digest := blob.Digest()
	if digest == blobaccess.BLOB_UNKNOWN_DIGEST {
		return "", fmt.Errorf("unknown digest")
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	b := a.blobs[digest.Encoded()]
	if b == nil {
		b, err := blob.Dup()
		if err != nil {
			return "", err
		}
		a.blobs[digest.Encoded()] = b
	}
	return digest.Encoded(), nil
}

func (a *Access) Close() error {
	list := errors.ErrorList{}
	for _, b := range a.blobs {
		list.Add(b.Close())
	}
	return list.Result()
}

var _ virtual.Access = (*Access)(nil)

type VersionAccess struct {
	access *Access
	comp   string
	vers   string
	desc   *compdesc.ComponentDescriptor
}

func (v *VersionAccess) GetDescriptor() *compdesc.ComponentDescriptor {
	return v.desc
}

func (v *VersionAccess) GetBlob(name string) (cpi.DataAccess, error) {
	return v.access.GetBlob(name)
}

func (v *VersionAccess) AddBlob(blob cpi.BlobAccess) (string, error) {
	return v.access.AddBlob(blob)
}

func (v *VersionAccess) Update() error {
	v.access.lock.Lock()
	defer v.access.lock.Unlock()

	if v.desc.GetName() != v.comp || v.desc.GetVersion() != v.vers {
		return errors.ErrInvalid(cpi.KIND_COMPONENTVERSION, common.VersionedElementKey(v.desc).String())
	}
	i := v.access.index.Get(v.comp, v.vers)
	if !reflect.DeepEqual(v.desc, i.CD()) {
		v.access.index.Set(v.desc, i.Info())
	}
	return nil
}

func (v *VersionAccess) Close() error {
	return v.Update()
}

func (v *VersionAccess) IsReadOnly() bool {
	return false
}

func (v *VersionAccess) GetInexpensiveContentVersionIdentity(a cpi.AccessSpec) string {
	switch a.GetKind() { //nolint:gocritic // to be extended
	case localblob.Type:
		return a.(*localblob.AccessSpec).LocalReference
	}
	return ""
}

var _ virtual.VersionAccess = (*VersionAccess)(nil)
