// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package repocpi

import (
	"io"
	"sync/atomic"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/utils"
)

// StorageBackendImpl is an interface which can be implemented
// to provide a complete repository view with repository, component
// and component version objects, which are generically implemented
// based on the methods of this interface.
//
// A repository interface based on this implementation interface can be
// created using the function NewStorageBackend.
type StorageBackendImpl interface {
	// repository related methods.

	io.Closer
	GetContext() cpi.Context
	GetSpecification() cpi.RepositorySpec
	IsReadOnly() bool

	ComponentLister() cpi.ComponentLister
	HasComponent(name string) (bool, error)
	HasComponentVersion(key common.NameVersion) (bool, error)

	// component related methods.

	ListVersions(comp string) ([]string, error)
	HasVersion(vers string) (bool, error)

	// version related methods.

	GetDescriptor(key common.NameVersion) (*compdesc.ComponentDescriptor, error)
	SetDescriptor(key common.NameVersion, descriptor *compdesc.ComponentDescriptor) error
	AccessMethod(key common.NameVersion, acc cpi.AccessSpec, cv refmgmt.ExtendedAllocatable) (cpi.AccessMethod, error)
	GetInexpensiveContentVersionIdentity(key common.NameVersion, acc cpi.AccessSpec, cv refmgmt.ExtendedAllocatable) string
	GetStorageContext(key common.NameVersion) cpi.StorageContext
	GetBlob(key common.NameVersion, name string) (cpi.DataAccess, error)
	AddBlob(key common.NameVersion, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error)
}

type storageBackendRepository struct {
	bridge RepositoryBridge
	closed atomic.Bool
	noref  cpi.Repository
	kind   string
	impl   StorageBackendImpl
}

var _ RepositoryImpl = (*storageBackendRepository)(nil)

// NewStorageBackend provides a complete repository view
// with repository, component and component version objects
// based on the implementation of the sole interface StorageBackendImpl.
// No further implementations are required besides a dedicated
// specification object, the dependent object
// types are generically provided based on the methods of this
// interface.
// The kind parameter is used to denote the kind of repository
// in ids and log messages.
func NewStorageBackend(kind string, impl StorageBackendImpl) cpi.Repository {
	backend := &storageBackendRepository{
		impl: impl,
		kind: kind,
	}
	return NewRepository(backend, kind)
}

func (s *storageBackendRepository) SetBridge(bridge RepositoryBridge) {
	s.bridge = bridge
	s.noref = NewNoneRefRepositoryView(bridge)
}

func (s *storageBackendRepository) Close() error {
	if s.closed.Swap(true) {
		return ErrClosed
	}
	return s.impl.Close()
}

func (s *storageBackendRepository) GetContext() cpi.Context {
	return s.impl.GetContext()
}

func (s *storageBackendRepository) GetSpecification() cpi.RepositorySpec {
	return s.impl.GetSpecification()
}

func (s *storageBackendRepository) ComponentLister() cpi.ComponentLister {
	return s.impl.ComponentLister()
}

func (s *storageBackendRepository) ExistsComponentVersion(name string, version string) (bool, error) {
	return s.impl.HasComponentVersion(common.NewNameVersion(name, version))
}

func (s *storageBackendRepository) LookupComponent(name string) (*ComponentAccessInfo, error) {
	if ok, err := s.impl.HasComponent(name); !ok || err != nil {
		return nil, err
	}
	impl := &storageBackendComponent{
		repo: s,
		name: name,
	}
	return &ComponentAccessInfo{
		Impl: impl,
		Kind: s.kind + " component",
		Main: true,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

type storageBackendComponent struct {
	bridge ComponentAccessBridge
	repo   *storageBackendRepository
	name   string
}

var _ ComponentAccessImpl = (*storageBackendComponent)(nil)

func (s *storageBackendComponent) SetBridge(bridge ComponentAccessBridge) {
	s.bridge = bridge
}

func (s *storageBackendComponent) GetParentBridge() RepositoryViewManager {
	return s.repo.bridge
}

func (s *storageBackendComponent) Close() error {
	return nil
}

func (s *storageBackendComponent) GetContext() cpi.Context {
	return s.repo.impl.GetContext()
}

func (s *storageBackendComponent) GetName() string {
	return s.name
}

func (s *storageBackendComponent) IsReadOnly() bool {
	return s.repo.impl.IsReadOnly()
}

func (s *storageBackendComponent) ListVersions() ([]string, error) {
	return s.repo.impl.ListVersions(s.name)
}

func (s *storageBackendComponent) HasVersion(vers string) (bool, error) {
	return s.repo.impl.HasVersion(s.name)
}

func (s *storageBackendComponent) LookupVersion(version string) (*ComponentVersionAccessInfo, error) {
	if ok, err := s.repo.impl.HasComponentVersion(common.NewNameVersion(s.name, version)); !ok || err != nil {
		return nil, err
	}

	name := common.NewNameVersion(s.name, version)
	d, err := s.repo.impl.GetDescriptor(name)
	if err != nil {
		return nil, err
	}

	impl := &storageBackendComponentVersion{
		comp:       s,
		name:       name,
		descriptor: d,
	}
	return &ComponentVersionAccessInfo{
		Impl:       impl,
		Lazy:       true,
		Persistent: true,
	}, nil
}

func (s *storageBackendComponent) NewVersion(version string, overwrite ...bool) (*ComponentVersionAccessInfo, error) {
	ok, err := s.repo.impl.HasComponentVersion(common.NewNameVersion(s.name, version))
	if err != nil {
		return nil, err
	}
	if ok && !utils.Optional(overwrite...) {
		return nil, errors.ErrAlreadyExists(cpi.KIND_COMPONENTVERSION, s.name+"/"+version)
	}

	name := common.NewNameVersion(s.name, version)
	d := compdesc.New(s.name, version)

	impl := &storageBackendComponentVersion{
		comp:       s,
		name:       name,
		descriptor: d,
	}
	return &ComponentVersionAccessInfo{
		Impl:       impl,
		Lazy:       true,
		Persistent: false,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

type storageBackendComponentVersion struct {
	bridge     ComponentVersionAccessBridge
	comp       *storageBackendComponent
	name       common.NameVersion
	descriptor *compdesc.ComponentDescriptor
}

var _ ComponentVersionAccessImpl = (*storageBackendComponentVersion)(nil)

func (s *storageBackendComponentVersion) Close() error {
	return nil
}

func (s *storageBackendComponentVersion) GetContext() cpi.Context {
	return s.comp.repo.impl.GetContext()
}

func (s *storageBackendComponentVersion) SetBridge(bridge ComponentVersionAccessBridge) {
	s.bridge = bridge
}

func (s *storageBackendComponentVersion) GetParentBridge() ComponentAccessBridge {
	return s.comp.bridge
}

func (s *storageBackendComponentVersion) Repository() cpi.Repository {
	return s.comp.repo.noref
}

func (s *storageBackendComponentVersion) IsReadOnly() bool {
	return s.comp.repo.impl.IsReadOnly()
}

func (s *storageBackendComponentVersion) GetDescriptor() *compdesc.ComponentDescriptor {
	d, err := s.comp.repo.impl.GetDescriptor(s.name)
	if err != nil {
		return nil // TODO: handler error
	}
	return d
}

func (s *storageBackendComponentVersion) SetDescriptor(descriptor *compdesc.ComponentDescriptor) error {
	err := s.comp.repo.impl.SetDescriptor(s.name, descriptor)
	if err != nil {
		return err
	}
	s.descriptor = descriptor
	return nil
}

func (s *storageBackendComponentVersion) AccessMethod(acc cpi.AccessSpec, cv refmgmt.ExtendedAllocatable) (cpi.AccessMethod, error) {
	return s.comp.repo.impl.AccessMethod(s.name, acc, cv)
}

func (s *storageBackendComponentVersion) GetInexpensiveContentVersionIdentity(acc cpi.AccessSpec, cv refmgmt.ExtendedAllocatable) string {
	return s.comp.repo.impl.GetInexpensiveContentVersionIdentity(s.name, acc, cv)
}

func (s storageBackendComponentVersion) GetStorageContext() cpi.StorageContext {
	return s.comp.repo.impl.GetStorageContext(s.name)
}

func (s storageBackendComponentVersion) GetBlob(name string) (cpi.DataAccess, error) {
	return s.comp.repo.impl.GetBlob(s.name, name)
}

func (s storageBackendComponentVersion) AddBlob(blob cpi.BlobAccess, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	return s.comp.repo.impl.AddBlob(s.name, blob, refName, global)
}
