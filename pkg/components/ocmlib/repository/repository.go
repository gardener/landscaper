// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/utils/tarutils"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/virtual"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Index = virtual.Index[any]

type ComponentAccess struct {
	lock               sync.Mutex
	descriptorProvider ComponentDescriptorProvider
	blobsFs            vfs.FileSystem
	index              *Index
}

func NewAccess(descriptorProvider ComponentDescriptorProvider, blobFs vfs.FileSystem) (*ComponentAccess, error) {
	a := &ComponentAccess{
		descriptorProvider: descriptorProvider,
		blobsFs:            blobFs,
	}
	err := a.Index()
	if err != nil {
		return nil, err
	}
	return a, nil
}

// Index adds all Component Descriptors in the descriptorProvider to the Repository Index (in other words, it makes them
// known to the repository).
// This function used to initialize the repository but can also be used to update the repository in case Component
// Descriptors were added to the descriptorProvider.
func (a *ComponentAccess) Index() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.index = virtual.NewIndex[any]()

	entries, err := a.descriptorProvider.List()
	if err != nil {
		return fmt.Errorf("error indexing repository: %w", err)
	}
	for _, cd := range entries {
		err = a.index.Add(cd, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *ComponentAccess) ComponentLister() cpi.ComponentLister {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.index
}

func (a *ComponentAccess) ExistsComponentVersion(name string, version string) (bool, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	e := a.index.Get(name, version)
	return e != nil, nil
}

func (a *ComponentAccess) ListVersions(comp string) ([]string, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.index.GetVersions(comp), nil
}

func (a *ComponentAccess) GetComponentVersion(comp, version string) (virtual.VersionAccess, error) {
	var cd *compdesc.ComponentDescriptor

	a.lock.Lock()
	defer a.lock.Unlock()

	i := a.index.Get(comp, version)
	if i == nil {
		return nil, errors.ErrNotFound(cpi.KIND_COMPONENTVERSION, common.NewNameVersion(comp, version).String())
	} else {
		cd = i.CD()
	}
	return &ComponentVersionAccess{a, cd.GetName(), cd.GetVersion(), cd.Copy()}, nil
}

func (a *ComponentAccess) IsReadOnly() bool {
	return true
}

func (a *ComponentAccess) Close() error {
	return nil
}

var _ virtual.Access = (*ComponentAccess)(nil)

type ComponentVersionAccess struct {
	access *ComponentAccess
	comp   string
	vers   string
	desc   *compdesc.ComponentDescriptor
}

func (v *ComponentVersionAccess) GetDescriptor() *compdesc.ComponentDescriptor {
	return v.desc
}

func (v *ComponentVersionAccess) GetBlob(name string) (cpi.DataAccess, error) {
	if v.access.blobsFs == nil {
		return nil, vfs.ErrNotExist
	}

	filepath := path.Join("/", name)

	if ok, err := vfs.IsDir(v.access.blobsFs, filepath); ok {
		tempfile, err := accessio.NewTempFile(osfs.New(), os.TempDir(), "TEMP_BLOB_DATA")
		if err != nil {
			return nil, err
		}
		err = tarutils.PackFsIntoTar(v.access.blobsFs, filepath, tempfile.Writer(), tarutils.TarFileSystemOptions{})
		if err != nil {
			return nil, err
		}
		return tempfile.AsBlob(mime.MIME_TAR), nil
	} else {
		if err != nil {
			return nil, err
		}

		if ok, err := vfs.FileExists(v.access.blobsFs, filepath); ok {
			return accessio.DataAccessForFile(v.access.blobsFs, filepath), nil
		} else {
			if err != nil {
				return nil, err
			}
			return nil, vfs.ErrNotExist
		}
	}
}

func (v *ComponentVersionAccess) AddBlob(blob cpi.BlobAccess) (string, error) {
	return "", accessio.ErrReadOnly
}

func (v *ComponentVersionAccess) Update() error {
	return nil
}

func (v *ComponentVersionAccess) Close() error {
	return nil
}

func (v *ComponentVersionAccess) IsReadOnly() bool {
	return true
}

func (v *ComponentVersionAccess) GetInexpensiveContentVersionIdentity(a cpi.AccessSpec) string {
	switch a.GetKind() { //nolint:gocritic // to be extended
	case localblob.Type:
		blob, err := v.GetBlob(a.(*localblob.AccessSpec).LocalReference)
		if err != nil {
			return ""
		}
		defer blob.Close()
		dig, err := accessio.Digest(blob)
		if err != nil {
			return ""
		}
		return dig.String()
	}
	return ""
}

var _ virtual.VersionAccess = (*ComponentVersionAccess)(nil)
