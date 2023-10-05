// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"strconv"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/errors"
)

type DummyComponentVersionAccess struct {
	Context Context
}

var _ ComponentVersionAccess = (*DummyComponentVersionAccess)(nil)

func (d *DummyComponentVersionAccess) GetContext() Context {
	return d.Context
}

func (d *DummyComponentVersionAccess) Close() error {
	return nil
}

func (d *DummyComponentVersionAccess) IsClosed() bool {
	return false
}

func (d *DummyComponentVersionAccess) Dup() (ComponentVersionAccess, error) {
	return d, nil
}

func (d *DummyComponentVersionAccess) AdjustSourceAccess(meta *internal.SourceMeta, acc compdesc.AccessSpec) error {
	panic("implement me")
}

func (c *DummyComponentVersionAccess) Repository() Repository {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) GetName() string {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) GetVersion() string {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) GetDescriptor() *compdesc.ComponentDescriptor {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) GetResources() []ResourceAccess {
	return nil
}

func (d *DummyComponentVersionAccess) GetResource(meta metav1.Identity) (ResourceAccess, error) {
	return nil, errors.ErrNotFound("resource", meta.String())
}

func (d *DummyComponentVersionAccess) GetResourceIndex(metav1.Identity) int {
	return -1
}

func (d *DummyComponentVersionAccess) GetResourceByIndex(i int) (ResourceAccess, error) {
	return nil, errors.ErrInvalid("resource index", strconv.Itoa(i))
}

func (d *DummyComponentVersionAccess) GetResourcesByName(name string, selectors ...compdesc.IdentitySelector) ([]ResourceAccess, error) {
	return nil, errors.ErrInvalid("resource", name)
}

func (d *DummyComponentVersionAccess) GetSources() []SourceAccess {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) GetSource(meta metav1.Identity) (SourceAccess, error) {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) GetSourceIndex(metav1.Identity) int {
	return -1
}

func (d *DummyComponentVersionAccess) GetSourceByIndex(i int) (SourceAccess, error) {
	return nil, errors.ErrInvalid("source index", strconv.Itoa(i))
}

func (d *DummyComponentVersionAccess) GetReference(meta metav1.Identity) (ComponentReference, error) {
	return ComponentReference{}, errors.ErrNotFound("reference", meta.String())
}

func (d *DummyComponentVersionAccess) GetReferenceIndex(metav1.Identity) int {
	return -1
}

func (d *DummyComponentVersionAccess) GetReferenceByIndex(i int) (ComponentReference, error) {
	return ComponentReference{}, errors.ErrInvalid("reference index", strconv.Itoa(i))
}

func (d *DummyComponentVersionAccess) AccessMethod(spec AccessSpec) (AccessMethod, error) {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) GetInexpensiveContentVersionIdentity(spec AccessSpec) string {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) AddBlob(blob BlobAccess, arttype, refName string, global AccessSpec) (AccessSpec, error) {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) SetResourceBlob(meta *ResourceMeta, blob BlobAccess, refname string, global AccessSpec, opts ...internal.ModificationOption) error {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) AdjustResourceAccess(meta *internal.ResourceMeta, acc compdesc.AccessSpec, opts ...internal.ModificationOption) error {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) SetResource(meta *ResourceMeta, spec compdesc.AccessSpec, opts ...internal.ModificationOption) error {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) SetSourceBlob(meta *SourceMeta, blob BlobAccess, refname string, global AccessSpec) error {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) SetSource(meta *SourceMeta, spec compdesc.AccessSpec) error {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) SetReference(ref *ComponentReference) error {
	panic("implement me")
}

func (d *DummyComponentVersionAccess) DiscardChanges() {
}

func (d *DummyComponentVersionAccess) IsPersistent() bool {
	return false
}

func (d *DummyComponentVersionAccess) GetResourcesByIdentitySelectors(selectors ...compdesc.IdentitySelector) ([]internal.ResourceAccess, error) {
	return nil, nil
}

func (d *DummyComponentVersionAccess) GetResourcesByResourceSelectors(selectors ...compdesc.ResourceSelector) ([]internal.ResourceAccess, error) {
	return nil, nil
}

func (d *DummyComponentVersionAccess) GetReferencesByName(name string, selectors ...compdesc.IdentitySelector) (compdesc.References, error) {
	return nil, nil
}

func (d *DummyComponentVersionAccess) GetReferencesByIdentitySelectors(selectors ...compdesc.IdentitySelector) (compdesc.References, error) {
	return nil, nil
}

func (d *DummyComponentVersionAccess) GetReferencesByReferenceSelectors(selectors ...compdesc.ReferenceSelector) (compdesc.References, error) {
	return nil, nil
}
