// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"fmt"
	"strconv"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/keepblobattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type ComponentVersionAccess struct {
	view accessio.CloserView // handle close and refs
	*componentVersionAccessImpl
}

var _ cpi.ComponentVersionAccess = (*ComponentVersionAccess)(nil)

func NewComponentVersionAccess(container ComponentVersionContainer, lazy bool, persistent bool) (*ComponentVersionAccess, error) {
	s := &componentVersionAccessImpl{
		lazy:           lazy,
		discardChanges: !persistent,
		base:           container,
	}
	s.refs = accessio.NewRefCloser(s, true)
	v, err := s.View(true)
	if err != nil {
		return nil, err
	}
	return &ComponentVersionAccess{view: v, componentVersionAccessImpl: s}, nil
}

// implemented by view
// the rest is directly taken from the artifact set implementation

func (s *ComponentVersionAccess) Dup() (cpi.ComponentVersionAccess, error) {
	return s.View(false)
}

func (s *ComponentVersionAccess) View(main ...bool) (cpi.ComponentVersionAccess, error) {
	if s.view.IsClosed() {
		return nil, accessio.ErrClosed
	}
	v, err := s.componentVersionAccessImpl.View(main...)
	if err != nil {
		return nil, err
	}
	return &ComponentVersionAccess{view: v, componentVersionAccessImpl: s.componentVersionAccessImpl}, nil
}

func (s *ComponentVersionAccess) Close() error {
	err := s.Update(true)
	if err != nil {
		s.view.Close()

		return UpdateComponentVersionContainerError{
			Original: err,
			Name:     s.base.GetDescriptor().ObjectMeta.GetName(),
			Version:  s.base.GetDescriptor().ObjectMeta.GetVersion(),
		}
	}

	return s.view.Close()
}

func (s *ComponentVersionAccess) IsClosed() bool {
	return s.view.IsClosed()
}

func (s *ComponentVersionAccess) EnablePersistence() {
	s.discardChanges = false
}

func (s *ComponentVersionAccess) AddBlob(blob cpi.BlobAccess, artType, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	return s.componentVersionAccessImpl.AddBlob(s, blob, artType, refName, global)
}

func (s *ComponentVersionAccess) AccessMethod(a cpi.AccessSpec) (cpi.AccessMethod, error) {
	return s.componentVersionAccessImpl.AccessMethod(s, a)
}

func (s *ComponentVersionAccess) SetSource(meta *cpi.SourceMeta, acc compdesc.AccessSpec) error {
	return s.componentVersionAccessImpl.SetSource(s, meta, acc)
}

func (s *ComponentVersionAccess) SetSourceBlob(meta *cpi.SourceMeta, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) error {
	return s.componentVersionAccessImpl.SetSourceBlob(s, meta, blob, refName, global)
}

func (s *ComponentVersionAccess) SetResource(meta *cpi.ResourceMeta, acc compdesc.AccessSpec) error {
	return s.componentVersionAccessImpl.SetResource(s, meta, acc)
}

func (s *ComponentVersionAccess) SetResourceBlob(meta *cpi.ResourceMeta, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) error {
	return s.componentVersionAccessImpl.SetResourceBlob(s, meta, blob, refName, global)
}

func (s *ComponentVersionAccess) AdjustResourceAccess(meta *cpi.ResourceMeta, acc compdesc.AccessSpec) error {
	return s.componentVersionAccessImpl.AdjustResourceAccess(s, meta, acc)
}

func (s *ComponentVersionAccess) GetResource(id metav1.Identity) (cpi.ResourceAccess, error) {
	r, err := s.GetDescriptor().GetResourceByIdentity(id)
	if err != nil {
		return nil, err
	}
	return newResourceAccess(s, r.Access, r.ResourceMeta), nil
}

func (s *ComponentVersionAccess) GetResourceByIndex(i int) (cpi.ResourceAccess, error) {
	if i < 0 || i > len(s.GetDescriptor().Resources) {
		return nil, errors.ErrInvalid("resource index", strconv.Itoa(i))
	}
	r := s.GetDescriptor().Resources[i]
	return newResourceAccess(s, r.Access, r.ResourceMeta), nil
}

func (s *ComponentVersionAccess) GetResourcesByName(name string, selectors ...compdesc.IdentitySelector) ([]cpi.ResourceAccess, error) {
	resources, err := s.GetDescriptor().GetResourcesByName(name, selectors...)
	if err != nil {
		return nil, err
	}

	result := []cpi.ResourceAccess{}
	for _, resource := range resources {
		result = append(result, newResourceAccess(s, resource.Access, resource.ResourceMeta))
	}
	return result, nil
}

func (s *ComponentVersionAccess) GetResources() []cpi.ResourceAccess {
	result := []cpi.ResourceAccess{}
	for _, r := range s.GetDescriptor().Resources {
		result = append(result, newResourceAccess(s, r.Access, r.ResourceMeta))
	}
	return result
}

func (s *ComponentVersionAccess) GetSource(id metav1.Identity) (cpi.SourceAccess, error) {
	r, err := s.GetDescriptor().GetSourceByIdentity(id)
	if err != nil {
		return nil, err
	}
	return newSourceAccess(s, r.Access, r.SourceMeta), nil
}

func (s *ComponentVersionAccess) GetSourceByIndex(i int) (cpi.SourceAccess, error) {
	if i < 0 || i > len(s.GetDescriptor().Sources) {
		return nil, errors.ErrInvalid("source index", strconv.Itoa(i))
	}
	r := s.base.GetDescriptor().Sources[i]
	return newSourceAccess(s, r.Access, r.SourceMeta), nil
}

func (s *ComponentVersionAccess) GetSources() []cpi.SourceAccess {
	result := []cpi.SourceAccess{}
	for _, r := range s.GetDescriptor().Sources {
		result = append(result, newSourceAccess(s, r.Access, r.SourceMeta))
	}
	return result
}

func (s *ComponentVersionAccess) GetReferences() compdesc.References {
	return s.GetDescriptor().References
}

func (s *ComponentVersionAccess) GetReference(id metav1.Identity) (cpi.ComponentReference, error) {
	return s.GetDescriptor().GetReferenceByIdentity(id)
}

func (s *ComponentVersionAccess) GetReferenceByIndex(i int) (cpi.ComponentReference, error) {
	if i < 0 || i > len(s.GetDescriptor().References) {
		return cpi.ComponentReference{}, errors.ErrInvalid("reference index", strconv.Itoa(i))
	}
	return s.GetDescriptor().References[i], nil
}

func (s *ComponentVersionAccess) GetReferencesByName(name string, selectors ...compdesc.IdentitySelector) (compdesc.References, error) {
	return s.GetDescriptor().GetReferencesByName(name, selectors...)
}

////////////////////////////////////////////////////////////////////////////////

type componentVersionAccessImpl struct {
	refs           accessio.ReferencableCloser
	lazy           bool
	discardChanges bool
	base           ComponentVersionContainer
}

func (a *componentVersionAccessImpl) View(main ...bool) (accessio.CloserView, error) {
	return a.refs.View(main...)
}

func (a *componentVersionAccessImpl) Close() error {
	return errors.ErrListf("closing access").Add(a.Update(true), a.base.Close()).Result()
}

func (c *componentVersionAccessImpl) Repository() cpi.Repository {
	return c.base.Repository()
}

func (a *componentVersionAccessImpl) IsReadOnly() bool {
	return a.base.IsReadOnly()
}

func (a *componentVersionAccessImpl) IsClosed() bool {
	return a.base.IsClosed()
}

func (a *componentVersionAccessImpl) GetContext() cpi.Context {
	return a.base.GetContext()
}

func (a *componentVersionAccessImpl) GetName() string {
	return a.base.GetDescriptor().GetName()
}

func (a *componentVersionAccessImpl) GetVersion() string {
	return a.base.GetDescriptor().GetVersion()
}

func (a *componentVersionAccessImpl) AddBlob(cv cpi.ComponentVersionAccess, blob cpi.BlobAccess, artType, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	if blob == nil {
		return nil, errors.New("a resource has to be defined")
	}
	storagectx := a.base.GetStorageContext(cv)
	h := a.GetContext().BlobHandlers().LookupHandler(storagectx.GetImplementationRepositoryType(), artType, blob.MimeType())
	if h != nil {
		acc, err := h.StoreBlob(blob, artType, refName, nil, storagectx)
		if err != nil {
			return nil, err
		}
		if acc != nil {
			if !keepblobattr.Get(a.GetContext()) || acc.IsLocal(a.GetContext()) {
				return acc, nil
			}
			global = acc
		}
	}
	return a.base.AddBlobFor(storagectx, blob, refName, global)
}

func (c *componentVersionAccessImpl) AccessMethod(cv cpi.ComponentVersionAccess, a cpi.AccessSpec) (cpi.AccessMethod, error) {
	if !a.IsLocal(c.base.GetContext()) {
		// fall back to original version
		return a.AccessMethod(cv)
	}
	return c.base.AccessMethod(a)
}

func (a *componentVersionAccessImpl) GetDescriptor() *compdesc.ComponentDescriptor {
	return a.base.GetDescriptor()
}

func (c *componentVersionAccessImpl) AdjustResourceAccess(cv cpi.ComponentVersionAccess, meta *cpi.ResourceMeta, acc compdesc.AccessSpec) error {
	if err := c.checkAccessSpec(cv, acc); err != nil {
		return AccessCheckError{
			Original: err,
			Name:     meta.GetName(),
			Version:  meta.GetVersion(),
			Type:     meta.GetType(),
		}
	}

	cd := c.GetDescriptor()
	if idx := cd.GetResourceIndex(meta); idx == -1 {
		return errors.ErrUnknown(cpi.KIND_RESOURCE, meta.GetIdentity(cd.Resources).String())
	} else {
		cd.Resources[idx].Access = acc
	}

	return c.Update(false)
}

func (c *componentVersionAccessImpl) checkAccessSpec(cv cpi.ComponentVersionAccess, acc compdesc.AccessSpec) error {
	if _, err := NewBaseAccess(cv, acc).AccessMethod(); err != nil {
		return fmt.Errorf("unable to get access method: %w", err)
	}
	return nil
}

func (c *componentVersionAccessImpl) SetResource(cv cpi.ComponentVersionAccess, meta *cpi.ResourceMeta, acc compdesc.AccessSpec) error {
	if err := c.checkAccessSpec(cv, acc); err != nil {
		return AccessCheckError{
			Original: err,
			Name:     meta.GetName(),
			Version:  meta.GetVersion(),
			Type:     meta.GetType(),
		}
	}

	res := &compdesc.Resource{
		ResourceMeta: *meta.Copy(),
		Access:       acc,
	}

	if res.Relation == metav1.LocalRelation {
		if res.Version == "" {
			res.Version = c.GetVersion()
		}
	}

	cd := c.GetDescriptor()
	if idx := cd.GetResourceIndex(meta); idx == -1 {
		cd.Resources = append(cd.Resources, *res)
		cd.Signatures = nil
	} else {
		if !cd.Resources[idx].ResourceMeta.HashEqual(&res.ResourceMeta) {
			cd.Signatures = nil
		}
		cd.Resources[idx] = *res
	}
	return c.Update(false)
}

func (c *componentVersionAccessImpl) SetSource(cv cpi.ComponentVersionAccess, meta *cpi.SourceMeta, acc compdesc.AccessSpec) error {
	if err := c.checkAccessSpec(cv, acc); err != nil {
		if !errors.IsErrUnknown(err) {
			return AccessCheckError{
				Original: err,
				Name:     meta.GetName(),
				Version:  meta.GetVersion(),
				Type:     meta.GetType(),
			}
		}
	}

	res := &compdesc.Source{
		SourceMeta: *meta.Copy(),
		Access:     acc,
	}

	if res.Version == "" {
		res.Version = c.GetVersion()
	}

	if idx := c.GetDescriptor().GetSourceIndex(meta); idx == -1 {
		c.GetDescriptor().Sources = append(c.GetDescriptor().Sources, *res)
	} else {
		c.GetDescriptor().Sources[idx] = *res
	}
	return c.Update(false)
}

// AddResource adds a blob resource to the current archive.
func (c *componentVersionAccessImpl) SetResourceBlob(cv cpi.ComponentVersionAccess, meta *cpi.ResourceMeta, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) error {
	Logger(c).Info("adding resource blob", "resource", meta.Name)
	acc, err := c.AddBlob(cv, blob, meta.Type, refName, global)
	if err != nil {
		return fmt.Errorf("unable to add blob (component %s:%s resource %s): %w", c.GetName(), c.GetVersion(), meta.GetName(), err)
	}

	if err := c.SetResource(cv, meta, acc); err != nil {
		return fmt.Errorf("unable to set resource: %w", err)
	}

	return nil
}

func (c *componentVersionAccessImpl) SetSourceBlob(cv cpi.ComponentVersionAccess, meta *cpi.SourceMeta, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) error {
	Logger(c).Info("adding source blob", "source", meta.Name)
	acc, err := c.AddBlob(cv, blob, meta.Type, refName, global)
	if err != nil {
		return fmt.Errorf("unable to add blob: (component %s:%s resource %s): %w", c.GetName(), c.GetVersion(), meta.GetName(), err)
	}

	if err := c.SetSource(cv, meta, acc); err != nil {
		return fmt.Errorf("unable to set source: %w", err)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (c *componentVersionAccessImpl) SetReference(ref *cpi.ComponentReference) error {
	if idx := c.GetDescriptor().GetComponentReferenceIndex(*ref); idx == -1 {
		c.GetDescriptor().References = append(c.GetDescriptor().References, *ref)
	} else {
		c.GetDescriptor().References[idx] = *ref
	}
	return c.Update(false)
}

func (a *componentVersionAccessImpl) DiscardChanges() {
	a.discardChanges = true
}

func (a *componentVersionAccessImpl) Update(final bool) error {
	if (final || !a.lazy) && !a.discardChanges {
		return a.base.Update()
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type BaseAccess struct {
	vers   cpi.ComponentVersionAccess
	access compdesc.AccessSpec
}

type baseAccess = BaseAccess

func NewBaseAccess(cv cpi.ComponentVersionAccess, acc compdesc.AccessSpec) *BaseAccess {
	return &BaseAccess{vers: cv, access: acc}
}

func (r *BaseAccess) ComponentVersion() cpi.ComponentVersionAccess {
	return r.vers
}

func (r *BaseAccess) Access() (cpi.AccessSpec, error) {
	return r.vers.GetContext().AccessSpecForSpec(r.access)
}

func (r *BaseAccess) AccessMethod() (cpi.AccessMethod, error) {
	spec, err := r.Access()
	if err != nil {
		return nil, err
	}
	if spec, err := r.vers.AccessMethod(spec); err != nil {
		return nil, err
	} else {
		return spec, nil
	}
}

////////////////////////////////////////////////////////////////////////////////

type ResourceAccess struct {
	*baseAccess
	meta cpi.ResourceMeta
}

var _ cpi.ResourceAccess = (*ResourceAccess)(nil)

func newResourceAccess(componentVersion cpi.ComponentVersionAccess, accessSpec compdesc.AccessSpec, meta cpi.ResourceMeta) *ResourceAccess {
	return &ResourceAccess{
		baseAccess: &BaseAccess{
			vers:   componentVersion,
			access: accessSpec,
		},
		meta: meta,
	}
}

func (r ResourceAccess) Meta() *cpi.ResourceMeta {
	return &r.meta
}

////////////////////////////////////////////////////////////////////////////////

type SourceAccess struct {
	*baseAccess
	meta cpi.SourceMeta
}

var _ cpi.SourceAccess = (*SourceAccess)(nil)

func newSourceAccess(componentVersion cpi.ComponentVersionAccess, accessSpec compdesc.AccessSpec, meta cpi.SourceMeta) *SourceAccess {
	return &SourceAccess{
		baseAccess: &BaseAccess{
			vers:   componentVersion,
			access: accessSpec,
		},
		meta: meta,
	}
}

func (r SourceAccess) Meta() *cpi.SourceMeta {
	return &r.meta
}
