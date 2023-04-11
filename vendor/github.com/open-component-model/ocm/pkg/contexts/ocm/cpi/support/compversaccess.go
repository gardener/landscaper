// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"fmt"
	"strconv"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type ComponentVersionAccess struct {
	view accessio.CloserView // handle close and refs
	*componentVersionAccessImpl
}

// implemented by view
// the rest is directly taken from the artifact set implementation

func (s *ComponentVersionAccess) Dup() (cpi.ComponentVersionAccess, error) {
	if s.view.IsClosed() {
		return nil, accessio.ErrClosed
	}
	return s.componentVersionAccessImpl.Dup()
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

////////////////////////////////////////////////////////////////////////////////

type componentVersionAccessImpl struct {
	refs           accessio.ReferencableCloser
	lazy           bool
	discardChanges bool
	base           ComponentVersionContainer
}

var _ cpi.ComponentVersionAccess = (*ComponentVersionAccess)(nil)

func NewComponentVersionAccess(container ComponentVersionContainer, lazy bool, persistent bool) (*ComponentVersionAccess, error) {
	s := &componentVersionAccessImpl{
		lazy:           lazy,
		discardChanges: !persistent,
		base:           container,
	}
	s.refs = accessio.NewRefCloser(s, true)
	return s.View(true)
}

func (a *componentVersionAccessImpl) Dup() (cpi.ComponentVersionAccess, error) {
	return a.View(false)
}

func (a *componentVersionAccessImpl) View(main ...bool) (*ComponentVersionAccess, error) {
	v, err := a.refs.View(main...)
	if err != nil {
		return nil, err
	}
	return &ComponentVersionAccess{view: v, componentVersionAccessImpl: a}, nil
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

func (a *componentVersionAccessImpl) AddBlob(blob cpi.BlobAccess, artType, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	if blob == nil {
		return nil, errors.New("a resource has to be defined")
	}
	storagectx := a.base.GetStorageContext(a)
	h := a.GetContext().BlobHandlers().LookupHandler(storagectx.GetImplementationRepositoryType(), artType, blob.MimeType())
	if h != nil {
		acc, err := h.StoreBlob(blob, artType, refName, nil, storagectx)
		if err != nil {
			return nil, err
		}
		if acc != nil {
			return acc, nil
		}
	}
	return a.base.AddBlobFor(storagectx, blob, refName, global)
}

func (c *componentVersionAccessImpl) AccessMethod(a cpi.AccessSpec) (cpi.AccessMethod, error) {
	if !a.IsLocal(c.base.GetContext()) {
		// fall back to original version
		return a.AccessMethod(c)
	}
	return c.base.AccessMethod(a)
}

func (a *componentVersionAccessImpl) GetDescriptor() *compdesc.ComponentDescriptor {
	return a.base.GetDescriptor()
}

func (a *componentVersionAccessImpl) GetResource(id metav1.Identity) (cpi.ResourceAccess, error) {
	r, err := a.base.GetDescriptor().GetResourceByIdentity(id)
	if err != nil {
		return nil, err
	}
	return newResourceAccess(a, r.Access, r.ResourceMeta), nil
}

func (a *componentVersionAccessImpl) GetResourceByIndex(i int) (cpi.ResourceAccess, error) {
	if i < 0 || i > len(a.base.GetDescriptor().Resources) {
		return nil, errors.ErrInvalid("resource index", strconv.Itoa(i))
	}
	r := a.base.GetDescriptor().Resources[i]
	return newResourceAccess(a, r.Access, r.ResourceMeta), nil
}

func (a *componentVersionAccessImpl) GetResourcesByName(name string, selectors ...compdesc.IdentitySelector) ([]cpi.ResourceAccess, error) {
	resources, err := a.base.GetDescriptor().GetResourcesByName(name, selectors...)
	if err != nil {
		return nil, err
	}

	result := []cpi.ResourceAccess{}
	for _, resource := range resources {
		result = append(result, newResourceAccess(a, resource.Access, resource.ResourceMeta))
	}
	return result, nil
}

func (a *componentVersionAccessImpl) GetResources() []cpi.ResourceAccess {
	result := []cpi.ResourceAccess{}
	for _, r := range a.GetDescriptor().Resources {
		result = append(result, newResourceAccess(a, r.Access, r.ResourceMeta))
	}
	return result
}

func (a *componentVersionAccessImpl) GetSource(id metav1.Identity) (cpi.SourceAccess, error) {
	r, err := a.base.GetDescriptor().GetSourceByIdentity(id)
	if err != nil {
		return nil, err
	}
	return newSourceAccess(a, r.Access, r.SourceMeta), nil
}

func (a *componentVersionAccessImpl) GetSourceByIndex(i int) (cpi.SourceAccess, error) {
	if i < 0 || i > len(a.base.GetDescriptor().Sources) {
		return nil, errors.ErrInvalid("source index", strconv.Itoa(i))
	}
	r := a.base.GetDescriptor().Sources[i]
	return newSourceAccess(a, r.Access, r.SourceMeta), nil
}

func (a *componentVersionAccessImpl) GetSources() []cpi.SourceAccess {
	result := []cpi.SourceAccess{}
	for _, r := range a.GetDescriptor().Sources {
		result = append(result, newSourceAccess(a, r.Access, r.SourceMeta))
	}
	return result
}

func (a *componentVersionAccessImpl) GetReference(id metav1.Identity) (cpi.ComponentReference, error) {
	return a.base.GetDescriptor().GetReferenceByIdentity(id)
}

func (a *componentVersionAccessImpl) GetReferenceByIndex(i int) (cpi.ComponentReference, error) {
	if i < 0 || i > len(a.base.GetDescriptor().References) {
		return cpi.ComponentReference{}, errors.ErrInvalid("reference index", strconv.Itoa(i))
	}
	return a.base.GetDescriptor().References[i], nil
}

func (c *componentVersionAccessImpl) getAccessSpec(acc compdesc.AccessSpec) (cpi.AccessSpec, error) {
	return c.GetContext().AccessSpecForSpec(acc)
}

func (c *componentVersionAccessImpl) getAccessMethod(acc compdesc.AccessSpec) (cpi.AccessMethod, error) {
	spec, err := c.getAccessSpec(acc)
	if err != nil {
		return nil, err
	}
	if spec, err := c.AccessMethod(spec); err != nil {
		return nil, err
	} else {
		return spec, nil
	}
}

func (c *componentVersionAccessImpl) AdjustResourceAccess(meta *cpi.ResourceMeta, acc compdesc.AccessSpec) error {
	if err := c.checkAccessSpec(acc); err != nil {
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

func (c *componentVersionAccessImpl) checkAccessSpec(acc compdesc.AccessSpec) error {
	if _, err := c.getAccessMethod(acc); err != nil {
		return fmt.Errorf("unable to get access method: %w", err)
	}

	return nil
}

func (c *componentVersionAccessImpl) SetResource(meta *cpi.ResourceMeta, acc compdesc.AccessSpec) error {
	if err := c.checkAccessSpec(acc); err != nil {
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

func (c *componentVersionAccessImpl) SetSource(meta *cpi.SourceMeta, acc compdesc.AccessSpec) error {
	if err := c.checkAccessSpec(acc); err != nil {
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
func (c *componentVersionAccessImpl) SetResourceBlob(meta *cpi.ResourceMeta, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) error {
	Logger(c).Info("adding resource blob", "resource", meta.Name)
	acc, err := c.AddBlob(blob, meta.Type, refName, global)
	if err != nil {
		return fmt.Errorf("unable to add blob: %w", err)
	}

	if err := c.SetResource(meta, acc); err != nil {
		return fmt.Errorf("unable to set resource: %w", err)
	}

	return nil
}

func (c *componentVersionAccessImpl) SetSourceBlob(meta *cpi.SourceMeta, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) error {
	Logger(c).Info("adding source blob", "source", meta.Name)
	acc, err := c.AddBlob(blob, meta.Type, refName, global)
	if err != nil {
		return fmt.Errorf("unable to add blob: %w", err)
	}

	if err := c.SetSource(meta, acc); err != nil {
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
	vers   *componentVersionAccessImpl
	access compdesc.AccessSpec
}

func (r *BaseAccess) Access() (cpi.AccessSpec, error) {
	return r.vers.getAccessSpec(r.access)
}

func (r *BaseAccess) AccessMethod() (cpi.AccessMethod, error) {
	return r.vers.getAccessMethod(r.access)
}

////////////////////////////////////////////////////////////////////////////////

type ResourceAccess struct {
	*BaseAccess
	meta cpi.ResourceMeta
}

var _ cpi.ResourceAccess = (*ResourceAccess)(nil)

func newResourceAccess(componentVersion *componentVersionAccessImpl, accessSpec compdesc.AccessSpec, meta cpi.ResourceMeta) *ResourceAccess {
	return &ResourceAccess{
		BaseAccess: &BaseAccess{
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
	*BaseAccess
	meta cpi.SourceMeta
}

var _ cpi.SourceAccess = (*SourceAccess)(nil)

func newSourceAccess(componentVersion *componentVersionAccessImpl, accessSpec compdesc.AccessSpec, meta cpi.SourceMeta) *SourceAccess {
	return &SourceAccess{
		BaseAccess: &BaseAccess{
			vers:   componentVersion,
			access: accessSpec,
		},
		meta: meta,
	}
}

func (r SourceAccess) Meta() *cpi.SourceMeta {
	return &r.meta
}
