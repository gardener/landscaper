// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"fmt"
	"io"
	"strconv"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessio/resource"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/keepblobattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
	"github.com/open-component-model/ocm/pkg/utils/selector"
)

// View objects are the user facing generic implementations of the context interfaces.
// They are responsible to handle the reference counting and use
// shared implementations objects for th concrete type-specific implementations.
// Additionally, they are used to implement interface functionality which is
// common to all implementations and NOT dependent on the backend system technology.

var ErrClosed = resource.ErrClosed

////////////////////////////////////////////////////////////////////////////////

type _RepositoryView interface {
	resource.ResourceViewInt[Repository] // here you have to redeclare
}

type RepositoryViewManager = resource.ViewManager[Repository] // here you have to use an alias

type RepositoryImpl interface {
	resource.ResourceImplementation[Repository]
	internal.RepositoryImpl
}

type _RepositoryImplBase = resource.ResourceImplBase[Repository]

type RepositoryImplBase struct {
	_RepositoryImplBase
	ctx Context
}

func (b *RepositoryImplBase) GetContext() Context {
	return b.ctx
}

func NewRepositoryImplBase(ctx Context, closer ...io.Closer) *RepositoryImplBase {
	base, _ := resource.NewResourceImplBase[Repository, io.Closer](nil, closer...)
	return &RepositoryImplBase{
		_RepositoryImplBase: *base,
		ctx:                 ctx,
	}
}

type repositoryView struct {
	_RepositoryView
	impl RepositoryImpl
}

var (
	_ Repository                           = (*repositoryView)(nil)
	_ credentials.ConsumerIdentityProvider = (*repositoryView)(nil)
)

func GetRepositoryImplementation(n Repository) (RepositoryImpl, error) {
	if v, ok := n.(*repositoryView); ok {
		return v.impl, nil
	}
	return nil, errors.ErrNotSupported("repository implementation type", fmt.Sprintf("%T", n))
}

func repositoryViewCreator(i RepositoryImpl, v resource.CloserView, d RepositoryViewManager) Repository {
	return &repositoryView{
		_RepositoryView: resource.NewView[Repository](v, d),
		impl:            i,
	}
}

// NewNoneRefRepositoryView provides a repository reflecting the state of the
// view manager without holding an additional reference.
func NewNoneRefRepositoryView(i RepositoryImpl) Repository {
	return &repositoryView{
		_RepositoryView: resource.NewView[Repository](resource.NewNonRefView[Repository](i), i),
		impl:            i,
	}
}

func NewRepository(impl RepositoryImpl, name ...string) Repository {
	return resource.NewResource[Repository](impl, repositoryViewCreator, utils.OptionalDefaulted("OCM repo", name...), true)
}

func (r *repositoryView) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	return credentials.GetProvidedConsumerId(r.impl, uctx...)
}

func (r *repositoryView) GetIdentityMatcher() string {
	return credentials.GetProvidedIdentityMatcher(r.impl)
}

func (r *repositoryView) GetSpecification() internal.RepositorySpec {
	return r.impl.GetSpecification()
}

func (r *repositoryView) GetContext() internal.Context {
	return r.impl.GetContext()
}

func (r *repositoryView) ComponentLister() internal.ComponentLister {
	return r.impl.ComponentLister()
}

func (r *repositoryView) ExistsComponentVersion(name string, version string) (ok bool, err error) {
	err = r.Execute(func() error {
		ok, err = r.impl.ExistsComponentVersion(name, version)
		return err
	})
	return ok, err
}

func (r *repositoryView) LookupComponentVersion(name string, version string) (acc ComponentVersionAccess, err error) {
	err = r.Execute(func() error {
		acc, err = r.impl.LookupComponentVersion(name, version)
		return err
	})
	return acc, err
}

func (r *repositoryView) LookupComponent(name string) (acc ComponentAccess, err error) {
	err = r.Execute(func() error {
		acc, err = r.impl.LookupComponent(name)
		return err
	})
	return acc, err
}

////////////////////////////////////////////////////////////////////////////////

type _ComponentAccessView interface {
	resource.ResourceViewInt[ComponentAccess] // here you have to redeclare
}

type ComponentAccessViewManager = resource.ViewManager[ComponentAccess] // here you have to use an alias

type ComponentAccessImpl interface {
	resource.ResourceImplementation[ComponentAccess]
	internal.ComponentAccessImpl

	IsReadOnly() bool
	GetName() string
}

type _ComponentAccessImplBase = resource.ResourceImplBase[ComponentAccess]

type ComponentAccessImplBase struct {
	*_ComponentAccessImplBase
	ctx  Context
	name string
}

func NewComponentAccessImplBase(ctx Context, name string, repo RepositoryViewManager, closer ...io.Closer) (*ComponentAccessImplBase, error) {
	base, err := resource.NewResourceImplBase[ComponentAccess](repo, closer...)
	if err != nil {
		return nil, err
	}
	return &ComponentAccessImplBase{
		_ComponentAccessImplBase: base,
		ctx:                      ctx,
		name:                     name,
	}, nil
}

func (b *ComponentAccessImplBase) GetContext() Context {
	return b.ctx
}

func (b *ComponentAccessImplBase) GetName() string {
	return b.name
}

type componentAccessView struct {
	_ComponentAccessView
	impl ComponentAccessImpl
}

var _ ComponentAccess = (*componentAccessView)(nil)

func GetComponentAccessImplementation(n ComponentAccess) (ComponentAccessImpl, error) {
	if v, ok := n.(*componentAccessView); ok {
		return v.impl, nil
	}
	return nil, errors.ErrNotSupported("component implementation type", fmt.Sprintf("%T", n))
}

func componentAccessViewCreator(i ComponentAccessImpl, v resource.CloserView, d ComponentAccessViewManager) ComponentAccess {
	return &componentAccessView{
		_ComponentAccessView: resource.NewView[ComponentAccess](v, d),
		impl:                 i,
	}
}

func NewComponentAccess(impl ComponentAccessImpl, kind ...string) ComponentAccess {
	return resource.NewResource[ComponentAccess](impl, componentAccessViewCreator, fmt.Sprintf("%s %s", utils.OptionalDefaulted("component", kind...), impl.GetName()), true)
}

func (c *componentAccessView) GetContext() Context {
	return c.impl.GetContext()
}

func (c *componentAccessView) GetName() string {
	return c.impl.GetName()
}

func (c *componentAccessView) ListVersions() (list []string, err error) {
	err = c.Execute(func() error {
		list, err = c.impl.ListVersions()
		return err
	})
	return list, err
}

func (c *componentAccessView) LookupVersion(version string) (acc ComponentVersionAccess, err error) {
	err = c.Execute(func() error {
		acc, err = c.impl.LookupVersion(version)
		return err
	})
	return acc, err
}

func (c *componentAccessView) AddVersion(acc ComponentVersionAccess) error {
	if acc.GetName() != c.GetName() {
		return errors.ErrInvalid("component name", acc.GetName())
	}
	return c.Execute(func() error {
		return c.impl.AddVersion(acc)
	})
}

func (c *componentAccessView) NewVersion(version string, overrides ...bool) (acc ComponentVersionAccess, err error) {
	err = c.Execute(func() error {
		if c.impl.IsReadOnly() {
			return accessio.ErrReadOnly
		}
		acc, err = c.impl.NewVersion(version, overrides...)
		return err
	})
	return acc, err
}

func (c *componentAccessView) HasVersion(vers string) (ok bool, err error) {
	err = c.Execute(func() error {
		ok, err = c.impl.HasVersion(vers)
		return err
	})
	return ok, err
}

////////////////////////////////////////////////////////////////////////////////

type _ComponentVersionAccessView interface {
	resource.ResourceViewInt[ComponentVersionAccess]
}

type ComponentVersionAccessViewManager = resource.ViewManager[ComponentVersionAccess]

type ComponentVersionAccessImpl interface {
	resource.ResourceImplementation[ComponentVersionAccess]
	internal.ComponentVersionAccessImpl

	AccessMethod(ComponentVersionAccess, AccessSpec) (AccessMethod, error)

	GetInexpensiveContentVersionIdentity(ComponentVersionAccess, AccessSpec) string

	// GetStorageContext creates a storage context for blobs
	// that is used to feed blob handlers for specific blob storage methods.
	// If no handler accepts the blob, the AddBlobFor method will
	// be used to store the blob
	GetStorageContext(cv ComponentVersionAccess) StorageContext

	// AddBlobFor stores a local blob together with the component and
	// potentially provides a global reference according to the OCI distribution spec
	// if the blob described an oci artifact.
	// The resulting access information (global and local) is provided as
	// an access method specification usable in a component descriptor.
	// This is the direct technical storage, without caring about any handler.
	AddBlobFor(storagectx StorageContext, blob BlobAccess, refName string, global AccessSpec) (AccessSpec, error)

	IsReadOnly() bool
	Update(final bool) error
}

type _ComponentVersionAccessImplBase = resource.ResourceImplBase[ComponentVersionAccess]

type ComponentVersionAccessImplBase struct {
	*_ComponentVersionAccessImplBase
	ctx     Context
	name    string
	version string
}

func NewComponentVersionAccessImplBase(ctx Context, name, version string, repo ComponentAccessViewManager, closer ...io.Closer) (*ComponentVersionAccessImplBase, error) {
	base, err := resource.NewResourceImplBase[ComponentVersionAccess](repo, closer...)
	if err != nil {
		return nil, err
	}
	return &ComponentVersionAccessImplBase{
		_ComponentVersionAccessImplBase: base,
		ctx:                             ctx,
		name:                            name,
		version:                         version,
	}, nil
}

func (b *ComponentVersionAccessImplBase) Close() error {
	return b._ComponentVersionAccessImplBase.Close() // just for debugger to catch Close on this type
}

func (b *ComponentVersionAccessImplBase) GetContext() Context {
	return b.ctx
}

func (b *ComponentVersionAccessImplBase) GetName() string {
	return b.name
}

func (b *ComponentVersionAccessImplBase) GetVersion() string {
	return b.version
}

type componentVersionAccessView struct {
	_ComponentVersionAccessView
	impl ComponentVersionAccessImpl
}

var _ ComponentVersionAccess = (*componentVersionAccessView)(nil)

func GetComponentVersionAccessImplementation(n ComponentVersionAccess) (ComponentVersionAccessImpl, error) {
	if v, ok := n.(*componentVersionAccessView); ok {
		return v.impl, nil
	}
	return nil, errors.ErrNotSupported("component version implementation type", fmt.Sprintf("%T", n))
}

func artifactAccessViewCreator(i ComponentVersionAccessImpl, v resource.CloserView, d resource.ViewManager[ComponentVersionAccess]) ComponentVersionAccess {
	return &componentVersionAccessView{
		_ComponentVersionAccessView: resource.NewView[ComponentVersionAccess](v, d),
		impl:                        i,
	}
}

func NewComponentVersionAccess(impl ComponentVersionAccessImpl) ComponentVersionAccess {
	return resource.NewResource[ComponentVersionAccess](impl, artifactAccessViewCreator, fmt.Sprintf("component version  %s/%s", impl.GetName(), impl.GetVersion()), true)
}

func (c *componentVersionAccessView) Repository() Repository {
	return c.impl.Repository()
}

func (c *componentVersionAccessView) GetContext() internal.Context {
	return c.impl.GetContext()
}

func (c *componentVersionAccessView) GetName() string {
	return c.impl.GetName()
}

func (c *componentVersionAccessView) GetVersion() string {
	return c.impl.GetVersion()
}

func (c *componentVersionAccessView) GetDescriptor() *compdesc.ComponentDescriptor {
	return c.impl.GetDescriptor()
}

func (c *componentVersionAccessView) AccessMethod(spec AccessSpec) (meth AccessMethod, err error) {
	err = c.Execute(func() error {
		if !spec.IsLocal(c.GetContext()) {
			// fall back to original version
			meth, err = spec.AccessMethod(c)
		} else {
			meth, err = c.impl.AccessMethod(c, spec)
		}
		return err
	})
	return meth, err
}

func (c *componentVersionAccessView) GetInexpensiveContentVersionIdentity(spec AccessSpec) string {
	var id string
	_ = c.Execute(func() error {
		if !spec.IsLocal(c.GetContext()) {
			// fall back to original version
			id = spec.GetInexpensiveContentVersionIdentity(c)
		} else {
			id = c.impl.GetInexpensiveContentVersionIdentity(c, spec)
		}
		return nil
	})
	return id
}

func (c *componentVersionAccessView) AddBlob(blob cpi.BlobAccess, artType, refName string, global AccessSpec) (AccessSpec, error) {
	if blob == nil {
		return nil, errors.New("a resource has to be defined")
	}
	if c.impl.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	storagectx := c.impl.GetStorageContext(c)
	h := c.GetContext().BlobHandlers().LookupHandler(storagectx.GetImplementationRepositoryType(), artType, blob.MimeType())
	if h != nil {
		acc, err := h.StoreBlob(blob, artType, refName, nil, storagectx)
		if err != nil {
			return nil, err
		}
		if acc != nil {
			if !keepblobattr.Get(c.GetContext()) || acc.IsLocal(c.GetContext()) {
				return acc, nil
			}
			global = acc
		}
	}
	return c.impl.AddBlobFor(storagectx, blob, refName, global)
}

func (c *componentVersionAccessView) AdjustResourceAccess(meta *ResourceMeta, acc compdesc.AccessSpec, opts ...internal.ModificationOption) error {
	cd := c.GetDescriptor()
	if idx := cd.GetResourceIndex(meta); idx == -1 {
		return errors.ErrUnknown(KIND_RESOURCE, meta.GetIdentity(cd.Resources).String())
	}
	return c.SetResource(meta, acc, opts...)
}

// SetResourceBlob adds a blob resource to the component version.
func (c *componentVersionAccessView) SetResourceBlob(meta *ResourceMeta, blob cpi.BlobAccess, refName string, global AccessSpec, opts ...internal.ModificationOption) error {
	Logger(c).Info("adding resource blob", "resource", meta.Name)
	acc, err := c.AddBlob(blob, meta.Type, refName, global)
	if err != nil {
		return fmt.Errorf("unable to add blob (component %s:%s resource %s): %w", c.GetName(), c.GetVersion(), meta.GetName(), err)
	}

	if err := c.SetResource(meta, acc, append(opts, internal.ModifyResource())...); err != nil {
		return fmt.Errorf("unable to set resource: %w", err)
	}

	return nil
}

func (c *componentVersionAccessView) AdjustSourceAccess(meta *SourceMeta, acc compdesc.AccessSpec) error {
	cd := c.GetDescriptor()
	if idx := cd.GetSourceIndex(meta); idx == -1 {
		return errors.ErrUnknown(KIND_RESOURCE, meta.GetIdentity(cd.Resources).String())
	}
	return c.SetSource(meta, acc)
}

func (c *componentVersionAccessView) SetSourceBlob(meta *SourceMeta, blob BlobAccess, refName string, global AccessSpec) error {
	Logger(c).Info("adding source blob", "source", meta.Name)
	acc, err := c.AddBlob(blob, meta.Type, refName, global)
	if err != nil {
		return fmt.Errorf("unable to add blob: (component %s:%s source %s): %w", c.GetName(), c.GetVersion(), meta.GetName(), err)
	}

	if err := c.SetSource(meta, acc); err != nil {
		return fmt.Errorf("unable to set source: %w", err)
	}

	return nil
}

func (c *componentVersionAccessView) SetResource(meta *internal.ResourceMeta, acc compdesc.AccessSpec, modopts ...internal.ModificationOption) error {
	if c.impl.IsReadOnly() {
		return accessio.ErrReadOnly
	}

	res := &compdesc.Resource{
		ResourceMeta: *meta.Copy(),
		Access:       acc,
	}

	ctx := c.impl.GetContext()
	opts := internal.NewModificationOptions(modopts...)
	CompleteModificationOptions(ctx, opts)

	spec, err := ctx.AccessSpecForSpec(acc)
	if err != nil {
		return err
	}
	meth, err := c.AccessMethod(spec)
	if err != nil {
		return err
	}
	defer meth.Close()

	return c.Execute(func() error {
		var old *compdesc.Resource

		if res.Relation == metav1.LocalRelation {
			if res.Version == "" {
				res.Version = c.GetVersion()
			}
		}

		cd := c.impl.GetDescriptor()
		idx := cd.GetResourceIndex(&res.ResourceMeta)
		if idx >= 0 {
			old = &cd.Resources[idx]
		}

		if old == nil {
			if !opts.IsModifyResource() && c.impl.IsPersistent() {
				return fmt.Errorf("new resource would invalidate signature")
			}
		}

		// evaluate given digesting constraints and settings
		hashAlgo, digester, digest := c.evaluateResourceDigest(res, old, *opts)
		hasher := opts.GetHasher(hashAlgo)
		if digester.HashAlgorithm == "" && hasher == nil {
			return errors.ErrUnknown(compdesc.KIND_HASH_ALGORITHM, hashAlgo)
		}

		if !compdesc.IsNoneAccessKind(res.Access.GetKind()) {
			var calculatedDigest *DigestDescriptor
			if (!opts.IsSkipVerify() && digest != "") || (!opts.IsSkipDigest() && digest == "") {
				dig, err := ctx.BlobDigesters().DetermineDigests(res.Type, hasher, opts.HasherProvider, meth, digester)
				if err != nil {
					return err
				}
				if len(dig) == 0 {
					return fmt.Errorf("%s: no digester accepts resource", res.Name)
				}
				calculatedDigest = &dig[0]
			}

			if digest != "" && !opts.IsSkipVerify() {
				if digest != calculatedDigest.Value {
					return fmt.Errorf("digest mismatch: %s != %s", calculatedDigest.Value, digest)
				}
			}

			if !opts.IsSkipDigest() {
				if digest == "" {
					res.Digest = calculatedDigest
				} else {
					res.Digest = &compdesc.DigestSpec{
						HashAlgorithm:          digester.HashAlgorithm,
						NormalisationAlgorithm: digester.NormalizationAlgorithm,
						Value:                  digest,
					}
				}
			}
		}

		if old != nil {
			eq := res.Equivalent(old)
			if !eq.IsLocalHashEqual() && !opts.IsModifyResource() && c.impl.IsPersistent() {
				return fmt.Errorf("resource would invalidate signature")
			}
			cd.Signatures = nil
		}

		if old == nil {
			cd.Resources = append(cd.Resources, *res)
		} else {
			cd.Resources[idx] = *res
		}
		return c.impl.Update(false)
	})
}

// evaluateResourceDigest evaluate given potentially partly set digest to determine defaults.
func (c *componentVersionAccessView) evaluateResourceDigest(res, old *compdesc.Resource, opts internal.ModificationOptions) (string, DigesterType, string) {
	var digester DigesterType

	hashAlgo := opts.DefaultHashAlgorithm
	value := ""
	if !res.Digest.IsNone() {
		if res.Digest.IsComplete() {
			value = res.Digest.Value
		}
		if res.Digest.HashAlgorithm != "" {
			hashAlgo = res.Digest.HashAlgorithm
		}
		if res.Digest.NormalisationAlgorithm != "" {
			digester = DigesterType{
				HashAlgorithm:          hashAlgo,
				NormalizationAlgorithm: res.Digest.NormalisationAlgorithm,
			}
		}
	}
	res.Digest = nil

	// keep potential old digest settings
	if old != nil && old.Type == res.Type {
		if !old.Digest.IsNone() {
			digester.HashAlgorithm = old.Digest.HashAlgorithm
			digester.NormalizationAlgorithm = old.Digest.NormalisationAlgorithm
			if opts.IsAcceptExistentDigests() && !opts.IsModifyResource() && c.impl.IsPersistent() {
				res.Digest = old.Digest
				value = old.Digest.Value
			}
		}
	}
	return hashAlgo, digester, value
}

func (c *componentVersionAccessView) SetSource(meta *internal.SourceMeta, acc compdesc.AccessSpec) error {
	if c.impl.IsReadOnly() {
		return accessio.ErrReadOnly
	}

	res := &compdesc.Source{
		SourceMeta: *meta.Copy(),
		Access:     acc,
	}
	return c.Execute(func() error {
		if res.Version == "" {
			res.Version = c.impl.GetVersion()
		}
		cd := c.impl.GetDescriptor()
		if idx := cd.GetSourceIndex(&res.SourceMeta); idx == -1 {
			cd.Sources = append(cd.Sources, *res)
		} else {
			cd.Sources[idx] = *res
		}
		return c.impl.Update(false)
	})
}

func (c *componentVersionAccessView) SetReference(ref *internal.ComponentReference) error {
	return c.Execute(func() error {
		cd := c.impl.GetDescriptor()
		if idx := cd.GetComponentReferenceIndex(*ref); idx == -1 {
			cd.References = append(cd.References, *ref)
		} else {
			cd.References[idx] = *ref
		}
		return c.impl.Update(false)
	})
}

func (c *componentVersionAccessView) DiscardChanges() {
	c.impl.DiscardChanges()
}

func (c *componentVersionAccessView) IsPersistent() bool {
	return c.impl.IsPersistent()
}

////////////////////////////////////////////////////////////////////////////////
// Standard Implementation for descriptor based methods

func (c *componentVersionAccessView) GetResource(id metav1.Identity) (ResourceAccess, error) {
	r, err := c.GetDescriptor().GetResourceByIdentity(id)
	if err != nil {
		return nil, err
	}
	return newResourceAccess(c, r.Access, r.ResourceMeta), nil
}

func (c *componentVersionAccessView) GetResourceIndex(id metav1.Identity) int {
	return c.GetDescriptor().GetResourceIndexByIdentity(id)
}

func (c *componentVersionAccessView) GetResourceByIndex(i int) (ResourceAccess, error) {
	if i < 0 || i >= len(c.GetDescriptor().Resources) {
		return nil, errors.ErrInvalid("resource index", strconv.Itoa(i))
	}
	r := c.GetDescriptor().Resources[i]
	return newResourceAccess(c, r.Access, r.ResourceMeta), nil
}

func (c *componentVersionAccessView) GetResourcesByName(name string, selectors ...compdesc.IdentitySelector) ([]ResourceAccess, error) {
	resources, err := c.GetDescriptor().GetResourcesByName(name, selectors...)
	if err != nil {
		return nil, err
	}

	result := []ResourceAccess{}
	for _, resource := range resources {
		result = append(result, newResourceAccess(c, resource.Access, resource.ResourceMeta))
	}
	return result, nil
}

func (c *componentVersionAccessView) GetResources() []ResourceAccess {
	result := []ResourceAccess{}
	for _, r := range c.GetDescriptor().Resources {
		result = append(result, newResourceAccess(c, r.Access, r.ResourceMeta))
	}
	return result
}

// GetResourcesByIdentitySelectors returns resources that match the given identity selectors.
func (c *componentVersionAccessView) GetResourcesByIdentitySelectors(selectors ...compdesc.IdentitySelector) ([]ResourceAccess, error) {
	return c.GetResourcesBySelectors(selectors, nil)
}

// GetResourcesByResourceSelectors returns resources that match the given resource selectors.
func (c *componentVersionAccessView) GetResourcesByResourceSelectors(selectors ...compdesc.ResourceSelector) ([]ResourceAccess, error) {
	return c.GetResourcesBySelectors(nil, selectors)
}

// GetResourcesBySelectors returns resources that match the given selector.
func (c *componentVersionAccessView) GetResourcesBySelectors(selectors []compdesc.IdentitySelector, resourceSelectors []compdesc.ResourceSelector) ([]ResourceAccess, error) {
	resources := make([]ResourceAccess, 0)
	rscs := c.GetDescriptor().Resources
	for i := range rscs {
		selctx := compdesc.NewResourceSelectionContext(i, rscs)
		if len(selectors) > 0 {
			ok, err := selector.MatchSelectors(selctx.Identity(), selectors...)
			if err != nil {
				return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
			}
			if !ok {
				continue
			}
		}
		ok, err := compdesc.MatchResourceByResourceSelector(selctx, resourceSelectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
		}
		if !ok {
			continue
		}
		r, err := c.GetResourceByIndex(i)
		if err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	if len(resources) == 0 {
		return resources, compdesc.NotFound
	}
	return resources, nil
}

func (c *componentVersionAccessView) GetSource(id metav1.Identity) (SourceAccess, error) {
	r, err := c.GetDescriptor().GetSourceByIdentity(id)
	if err != nil {
		return nil, err
	}
	return newSourceAccess(c, r.Access, r.SourceMeta), nil
}

func (c *componentVersionAccessView) GetSourceIndex(id metav1.Identity) int {
	return c.GetDescriptor().GetSourceIndexByIdentity(id)
}

func (c *componentVersionAccessView) GetSourceByIndex(i int) (SourceAccess, error) {
	if i < 0 || i >= len(c.GetDescriptor().Sources) {
		return nil, errors.ErrInvalid("source index", strconv.Itoa(i))
	}
	r := c.GetDescriptor().Sources[i]
	return newSourceAccess(c, r.Access, r.SourceMeta), nil
}

func (c *componentVersionAccessView) GetSources() []SourceAccess {
	result := []SourceAccess{}
	for _, r := range c.GetDescriptor().Sources {
		result = append(result, newSourceAccess(c, r.Access, r.SourceMeta))
	}
	return result
}

func (c *componentVersionAccessView) GetReferences() compdesc.References {
	return c.GetDescriptor().References
}

func (c *componentVersionAccessView) GetReference(id metav1.Identity) (ComponentReference, error) {
	return c.GetDescriptor().GetReferenceByIdentity(id)
}

func (c *componentVersionAccessView) GetReferenceIndex(id metav1.Identity) int {
	return c.GetDescriptor().GetReferenceIndexByIdentity(id)
}

func (c *componentVersionAccessView) GetReferenceByIndex(i int) (ComponentReference, error) {
	if i < 0 || i > len(c.GetDescriptor().References) {
		return ComponentReference{}, errors.ErrInvalid("reference index", strconv.Itoa(i))
	}
	return c.GetDescriptor().References[i], nil
}

func (c *componentVersionAccessView) GetReferencesByName(name string, selectors ...compdesc.IdentitySelector) (compdesc.References, error) {
	return c.GetDescriptor().GetReferencesByName(name, selectors...)
}

// GetReferencesByIdentitySelectors returns references that match the given identity selectors.
func (c *componentVersionAccessView) GetReferencesByIdentitySelectors(selectors ...compdesc.IdentitySelector) (compdesc.References, error) {
	return c.GetReferencesBySelectors(selectors, nil)
}

// GetReferencesByReferenceSelectors returns references that match the given resource selectors.
func (c *componentVersionAccessView) GetReferencesByReferenceSelectors(selectors ...compdesc.ReferenceSelector) (compdesc.References, error) {
	return c.GetReferencesBySelectors(nil, selectors)
}

// GetReferencesBySelectors returns references that match the given selector.
func (c *componentVersionAccessView) GetReferencesBySelectors(selectors []compdesc.IdentitySelector, referenceSelectors []compdesc.ReferenceSelector) (compdesc.References, error) {
	references := make(compdesc.References, 0)
	refs := c.GetDescriptor().References
	for i := range refs {
		selctx := compdesc.NewReferenceSelectionContext(i, refs)
		if len(selectors) > 0 {
			ok, err := selector.MatchSelectors(selctx.Identity(), selectors...)
			if err != nil {
				return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
			}
			if !ok {
				continue
			}
		}
		ok, err := compdesc.MatchReferencesByReferenceSelector(selctx, referenceSelectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
		}
		if !ok {
			continue
		}
		references = append(references, *selctx.ComponentReference)
	}
	if len(references) == 0 {
		return references, compdesc.NotFound
	}
	return references, nil
}
