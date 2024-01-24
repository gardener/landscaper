// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package repocpi

import (
	"fmt"
	"io"
	"strconv"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/compose"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/plugin/descriptor"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/refmgmt/resource"
	"github.com/open-component-model/ocm/pkg/utils"
	"github.com/open-component-model/ocm/pkg/utils/selector"
)

// View objects are the user facing generic implementations of the context interfaces.
// They are responsible to handle the reference counting and use
// shared implementations objects for th concrete type-specific implementations.
// Additionally, they are used to implement interface functionality which is
// common to all implementations and NOT dependent on the backend system technology.

// here are the views implementing the user facing ComponentVersionAccess
// interface.

type _componentVersionAccessView interface {
	resource.ResourceViewInt[cpi.ComponentVersionAccess]
}

type ComponentVersionAccessViewManager = resource.ViewManager[cpi.ComponentVersionAccess]

type ComponentVersionAccessBridge interface {
	resource.ResourceImplementation[cpi.ComponentVersionAccess]
	common.VersionedElement
	io.Closer

	GetContext() cpi.Context
	Repository() cpi.Repository

	GetImplementation() ComponentVersionAccessImpl

	EnablePersistence() bool
	DiscardChanges()
	IsPersistent() bool

	GetDescriptor() *compdesc.ComponentDescriptor

	AccessMethod(cpi.AccessSpec, refmgmt.ExtendedAllocatable) (cpi.AccessMethod, error)
	GetInexpensiveContentVersionIdentity(cpi.AccessSpec, refmgmt.ExtendedAllocatable) string

	// GetStorageContext creates a storage context for blobs
	// that is used to feed blob handlers for specific blob storage methods.
	// If no handler accepts the blob, the AddBlob method will
	// be used to store the blob
	GetStorageContext() cpi.StorageContext

	// AddBlob stores a local blob together with the component and
	// potentially provides a global reference.
	// The resulting access information (global and local) is provided as
	// an access method specification usable in a component descriptor.
	// This is the direct technical storage, without caring about any handler.
	AddBlob(blob cpi.BlobAccess, arttype, refName string, global cpi.AccessSpec, final bool, opts *cpi.BlobUploadOptions) (cpi.AccessSpec, error)

	IsReadOnly() bool

	// ShouldUpdate checks, whether an update is indicated
	// by the state of object, considering persistence, lazy, discard
	// and update mode state
	ShouldUpdate(final bool) bool

	// GetBlobCache retieves the blob cache used to store preliminary
	// blob accesses for freshly generated local access specs not directly
	// usable until a component version is finally added to the repository.
	GetBlobCache() BlobCache

	// UseDirectAccess returns true if composition should be directly
	// forwarded to the repository backend.,
	UseDirectAccess() bool

	// Update persists the current state of the component version to the
	// underlying repository backend.
	Update(final bool) error
}

type componentVersionAccessView struct {
	_componentVersionAccessView
	bridge ComponentVersionAccessBridge
	err    error
}

var (
	_ cpi.ComponentVersionAccess = (*componentVersionAccessView)(nil)
	_ utils.Unwrappable          = (*componentVersionAccessView)(nil)
)

func GetComponentVersionAccessBridge(n cpi.ComponentVersionAccess) (ComponentVersionAccessBridge, error) {
	if v, ok := n.(*componentVersionAccessView); ok {
		return v.bridge, nil
	}
	return nil, errors.ErrNotSupported("component version bridge type", fmt.Sprintf("%T", n))
}

func GetComponentVersionAccessImplementation(n cpi.ComponentVersionAccess) (ComponentVersionAccessImpl, error) {
	if v, ok := n.(*componentVersionAccessView); ok {
		if b, ok := v.bridge.(*componentVersionAccessBridge); ok {
			return b.impl, nil
		}
		return nil, errors.ErrNotSupported("component version bridge type", fmt.Sprintf("%T", v.bridge))
	}
	return nil, errors.ErrNotSupported("component version implementation type", fmt.Sprintf("%T", n))
}

func artifactAccessViewCreator(i ComponentVersionAccessBridge, v resource.CloserView, d resource.ViewManager[cpi.ComponentVersionAccess]) cpi.ComponentVersionAccess {
	cv := &componentVersionAccessView{
		_componentVersionAccessView: resource.NewView[cpi.ComponentVersionAccess](v, d),
		bridge:                      i,
	}
	return cv
}

func NewComponentVersionAccess(name, version string, impl ComponentVersionAccessImpl, lazy, persistent, direct bool, closer ...io.Closer) (cpi.ComponentVersionAccess, error) {
	bridge, err := newComponentVersionAccessBridge(name, version, impl, lazy, persistent, direct, closer...)
	if err != nil {
		return nil, errors.Join(err, impl.Close())
	}
	return resource.NewResource[cpi.ComponentVersionAccess](bridge, artifactAccessViewCreator, fmt.Sprintf("component version  %s/%s", name, version), true), nil
}

func (c *componentVersionAccessView) Unwrap() interface{} {
	return c.bridge
}

func (c *componentVersionAccessView) Close() error {
	list := errors.ErrListf("closing %s", common.VersionedElementKey(c))
	err := c._componentVersionAccessView.Close()
	return list.Add(c.err, err).Result()
}

func (c *componentVersionAccessView) Repository() cpi.Repository {
	return c.bridge.Repository()
}

func (c *componentVersionAccessView) GetContext() internal.Context {
	return c.bridge.GetContext()
}

func (c *componentVersionAccessView) GetName() string {
	return c.bridge.GetName()
}

func (c *componentVersionAccessView) GetVersion() string {
	return c.bridge.GetVersion()
}

func (c *componentVersionAccessView) GetDescriptor() *compdesc.ComponentDescriptor {
	return c.bridge.GetDescriptor()
}

func (c *componentVersionAccessView) GetProvider() *compdesc.Provider {
	return c.GetDescriptor().Provider.Copy()
}

func (c *componentVersionAccessView) SetProvider(p *compdesc.Provider) error {
	return c.Execute(func() error {
		c.GetDescriptor().Provider = *p.Copy()
		return nil
	})
}

func (c *componentVersionAccessView) AccessMethod(spec cpi.AccessSpec) (meth cpi.AccessMethod, err error) {
	spec, err = c.GetContext().AccessSpecForSpec(spec)
	if err != nil {
		return nil, err
	}
	err = c.Execute(func() error {
		var err error
		meth, err = c.accessMethod(spec)
		return err
	})
	return meth, err
}

func (c *componentVersionAccessView) accessMethod(spec cpi.AccessSpec) (meth cpi.AccessMethod, err error) {
	switch {
	case spec.IsLocal(c.GetContext()):
		return c.bridge.AccessMethod(spec, c.Allocatable())
	default:
		return spec.AccessMethod(c)
	}
}

func (c *componentVersionAccessView) GetInexpensiveContentVersionIdentity(spec cpi.AccessSpec) string {
	var err error

	spec, err = c.GetContext().AccessSpecForSpec(spec)
	if err != nil {
		return ""
	}

	var id string
	_ = c.Execute(func() error {
		id = c.getInexpensiveContentVersionIdentity(spec)
		return nil
	})
	return id
}

func (c *componentVersionAccessView) getInexpensiveContentVersionIdentity(spec cpi.AccessSpec) string {
	switch {
	case compose.Is(spec):
		fallthrough
	case !spec.IsLocal(c.GetContext()):
		// fall back to original version
		return spec.GetInexpensiveContentVersionIdentity(c)
	default:
		return c.bridge.GetInexpensiveContentVersionIdentity(spec, c.Allocatable())
	}
}

func (c *componentVersionAccessView) Update() error {
	return c.Execute(func() error {
		if !c.bridge.IsPersistent() {
			return ErrTempVersion
		}
		return c.bridge.Update(true)
	})
}

func (c *componentVersionAccessView) AddBlob(blob cpi.BlobAccess, artType, refName string, global cpi.AccessSpec, opts ...internal.BlobUploadOption) (cpi.AccessSpec, error) {
	var spec cpi.AccessSpec
	eff := cpi.NewBlobUploadOptions(opts...)
	err := c.Execute(func() error {
		var err error
		spec, err = c.bridge.AddBlob(blob, artType, refName, global, false, eff)
		return err
	})

	return spec, err
}

func (c *componentVersionAccessView) AdjustResourceAccess(meta *cpi.ResourceMeta, acc compdesc.AccessSpec, opts ...internal.ModificationOption) error {
	cd := c.GetDescriptor()
	if idx := cd.GetResourceIndex(meta); idx >= 0 {
		return c.SetResource(&cd.Resources[idx].ResourceMeta, acc, opts...)
	}
	return errors.ErrUnknown(cpi.KIND_RESOURCE, meta.GetIdentity(cd.Resources).String())
}

// SetResourceBlob adds a blob resource to the component version.
func (c *componentVersionAccessView) SetResourceBlob(meta *cpi.ResourceMeta, blob cpi.BlobAccess, refName string, global cpi.AccessSpec, opts ...internal.BlobModificationOption) error {
	cpi.Logger(c).Debug("adding resource blob", "resource", meta.Name)
	if err := utils.ValidateObject(blob); err != nil {
		return err
	}
	eff := cpi.NewBlobModificationOptions(opts...)
	acc, err := c.AddBlob(blob, meta.Type, refName, global, eff)
	if err != nil {
		return fmt.Errorf("unable to add blob (component %s:%s resource %s): %w", c.GetName(), c.GetVersion(), meta.GetName(), err)
	}

	if err := c.SetResource(meta, acc, eff, cpi.ModifyResource()); err != nil {
		return fmt.Errorf("unable to set resource: %w", err)
	}

	return nil
}

func (c *componentVersionAccessView) AdjustSourceAccess(meta *cpi.SourceMeta, acc compdesc.AccessSpec) error {
	cd := c.GetDescriptor()
	if idx := cd.GetSourceIndex(meta); idx >= 0 {
		return c.SetSource(&cd.Sources[idx].SourceMeta, acc)
	}
	return errors.ErrUnknown(cpi.KIND_RESOURCE, meta.GetIdentity(cd.Resources).String())
}

func (c *componentVersionAccessView) SetSourceBlob(meta *cpi.SourceMeta, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) error {
	cpi.Logger(c).Debug("adding source blob", "source", meta.Name)
	if err := utils.ValidateObject(blob); err != nil {
		return err
	}
	acc, err := c.AddBlob(blob, meta.Type, refName, global)
	if err != nil {
		return fmt.Errorf("unable to add blob: (component %s:%s source %s): %w", c.GetName(), c.GetVersion(), meta.GetName(), err)
	}

	if err := c.SetSource(meta, acc); err != nil {
		return fmt.Errorf("unable to set source: %w", err)
	}

	return nil
}

func setAccess[T any, A internal.ArtifactAccess[T]](c *componentVersionAccessView, kind string, art A,
	set func(*T, compdesc.AccessSpec) error,
	setblob func(*T, cpi.BlobAccess, string, cpi.AccessSpec) error,
) error {
	if c.bridge.IsReadOnly() {
		return accessio.ErrReadOnly
	}
	meta := art.Meta()
	if meta == nil {
		return errors.Newf("no meta data provided by %s access", kind)
	}
	acc, err := art.Access()
	if err != nil && !errors.IsErrNotFoundElem(err, "", descriptor.KIND_ACCESSMETHOD) {
		return err
	}

	var (
		blob   cpi.BlobAccess
		hint   string
		global cpi.AccessSpec
	)

	if acc != nil {
		if !acc.IsLocal(c.GetContext()) {
			return set(meta, acc)
		}

		blob, err = accspeccpi.BlobAccessForAccessSpec(acc, c)
		if err != nil && errors.IsErrNotFoundElem(err, "", blobaccess.KIND_BLOB) {
			return err
		}
		hint = cpi.ReferenceHint(acc, c)
		global = cpi.GlobalAccess(acc, c.GetContext())
	}
	if blob == nil {
		blob, err = art.BlobAccess()
		if err != nil {
			return err
		}
		defer blob.Close()
	}
	if blob == nil {
		return errors.Newf("neither access nor blob specified in %s access", kind)
	}
	if v := art.ReferenceHint(); v != "" {
		hint = v
	}
	if v := art.GlobalAccess(); v != nil {
		global = v
	}
	return setblob(meta, blob, hint, global)
}

func (c *componentVersionAccessView) SetResourceAccess(art cpi.ResourceAccess, modopts ...cpi.BlobModificationOption) error {
	return setAccess(c, "resource", art,
		func(meta *cpi.ResourceMeta, acc compdesc.AccessSpec) error {
			return c.SetResource(meta, acc, cpi.NewBlobModificationOptions(modopts...))
		},
		func(meta *cpi.ResourceMeta, blob cpi.BlobAccess, hint string, global cpi.AccessSpec) error {
			return c.SetResourceBlob(meta, blob, hint, global, modopts...)
		})
}

func (c *componentVersionAccessView) SetResource(meta *internal.ResourceMeta, acc compdesc.AccessSpec, modopts ...cpi.ModificationOption) error {
	if c.bridge.IsReadOnly() {
		return accessio.ErrReadOnly
	}

	res := &compdesc.Resource{
		ResourceMeta: *meta.Copy(),
		Access:       acc,
	}

	ctx := c.bridge.GetContext()
	opts := internal.NewModificationOptions(modopts...)
	cpi.CompleteModificationOptions(ctx, opts)

	spec, err := c.bridge.GetContext().AccessSpecForSpec(acc)
	if err != nil {
		return err
	}

	// if the blob described by the access spec has been added
	// as local blob, just reuse the stored blob access
	// to calculate the digest to circumvent credential problems
	// for access specs generated by an uploader.
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

		cd := c.bridge.GetDescriptor()
		idx := cd.GetResourceIndex(&res.ResourceMeta)
		if idx >= 0 {
			old = &cd.Resources[idx]
		}

		if old == nil {
			if !opts.IsModifyResource() && c.bridge.IsPersistent() {
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
			var calculatedDigest *cpi.DigestDescriptor
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
			if !eq.IsLocalHashEqual() && c.bridge.IsPersistent() {
				if !opts.IsModifyResource() {
					return fmt.Errorf("resource would invalidate signature")
				}
				cd.Signatures = nil
			}
		}

		if old == nil {
			cd.Resources = append(cd.Resources, *res)
		} else {
			cd.Resources[idx] = *res
		}
		return c.bridge.Update(false)
	})
}

// evaluateResourceDigest evaluate given potentially partly set digest to determine defaults.
func (c *componentVersionAccessView) evaluateResourceDigest(res, old *compdesc.Resource, opts cpi.ModificationOptions) (string, cpi.DigesterType, string) {
	var digester cpi.DigesterType

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
			digester = cpi.DigesterType{
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
			if opts.IsAcceptExistentDigests() && !opts.IsModifyResource() && c.bridge.IsPersistent() {
				res.Digest = old.Digest
				value = old.Digest.Value
			}
		}
	}
	return hashAlgo, digester, value
}

func (c *componentVersionAccessView) SetSourceByAccess(art cpi.SourceAccess) error {
	return setAccess(c, "source", art,
		c.SetSource, c.SetSourceBlob)
}

func (c *componentVersionAccessView) SetSource(meta *cpi.SourceMeta, acc compdesc.AccessSpec) error {
	if c.bridge.IsReadOnly() {
		return accessio.ErrReadOnly
	}

	res := &compdesc.Source{
		SourceMeta: *meta.Copy(),
		Access:     acc,
	}
	return c.Execute(func() error {
		if res.Version == "" {
			res.Version = c.bridge.GetVersion()
		}
		cd := c.bridge.GetDescriptor()
		if idx := cd.GetSourceIndex(&res.SourceMeta); idx == -1 {
			cd.Sources = append(cd.Sources, *res)
		} else {
			cd.Sources[idx] = *res
		}
		return c.bridge.Update(false)
	})
}

func (c *componentVersionAccessView) SetReference(ref *cpi.ComponentReference) error {
	return c.Execute(func() error {
		cd := c.bridge.GetDescriptor()
		if idx := cd.GetComponentReferenceIndex(*ref); idx == -1 {
			cd.References = append(cd.References, *ref)
		} else {
			cd.References[idx] = *ref
		}
		return c.bridge.Update(false)
	})
}

func (c *componentVersionAccessView) DiscardChanges() {
	c.bridge.DiscardChanges()
}

func (c *componentVersionAccessView) IsPersistent() bool {
	return c.bridge.IsPersistent()
}

func (c *componentVersionAccessView) UseDirectAccess() bool {
	return c.bridge.UseDirectAccess()
}

////////////////////////////////////////////////////////////////////////////////
// Standard Implementation for descriptor based methods

func (c *componentVersionAccessView) GetResource(id metav1.Identity) (cpi.ResourceAccess, error) {
	r, err := c.GetDescriptor().GetResourceByIdentity(id)
	if err != nil {
		return nil, err
	}
	return cpi.NewResourceAccess(c, r.Access, r.ResourceMeta), nil
}

func (c *componentVersionAccessView) GetResourceIndex(id metav1.Identity) int {
	return c.GetDescriptor().GetResourceIndexByIdentity(id)
}

func (c *componentVersionAccessView) GetResourceByIndex(i int) (cpi.ResourceAccess, error) {
	if i < 0 || i >= len(c.GetDescriptor().Resources) {
		return nil, errors.ErrInvalid("resource index", strconv.Itoa(i))
	}
	r := c.GetDescriptor().Resources[i]
	return cpi.NewResourceAccess(c, r.Access, r.ResourceMeta), nil
}

func (c *componentVersionAccessView) GetResourcesByName(name string, selectors ...compdesc.IdentitySelector) ([]cpi.ResourceAccess, error) {
	resources, err := c.GetDescriptor().GetResourcesByName(name, selectors...)
	if err != nil {
		return nil, err
	}

	result := []cpi.ResourceAccess{}
	for _, resource := range resources {
		result = append(result, cpi.NewResourceAccess(c, resource.Access, resource.ResourceMeta))
	}
	return result, nil
}

func (c *componentVersionAccessView) GetResources() []cpi.ResourceAccess {
	result := []cpi.ResourceAccess{}
	for _, r := range c.GetDescriptor().Resources {
		result = append(result, cpi.NewResourceAccess(c, r.Access, r.ResourceMeta))
	}
	return result
}

// GetResourcesByIdentitySelectors returns resources that match the given identity selectors.
func (c *componentVersionAccessView) GetResourcesByIdentitySelectors(selectors ...compdesc.IdentitySelector) ([]cpi.ResourceAccess, error) {
	return c.GetResourcesBySelectors(selectors, nil)
}

// GetResourcesByResourceSelectors returns resources that match the given resource selectors.
func (c *componentVersionAccessView) GetResourcesByResourceSelectors(selectors ...compdesc.ResourceSelector) ([]cpi.ResourceAccess, error) {
	return c.GetResourcesBySelectors(nil, selectors)
}

// GetResourcesBySelectors returns resources that match the given selector.
func (c *componentVersionAccessView) GetResourcesBySelectors(selectors []compdesc.IdentitySelector, resourceSelectors []compdesc.ResourceSelector) ([]cpi.ResourceAccess, error) {
	resources := make([]cpi.ResourceAccess, 0)
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

func (c *componentVersionAccessView) GetSource(id metav1.Identity) (cpi.SourceAccess, error) {
	r, err := c.GetDescriptor().GetSourceByIdentity(id)
	if err != nil {
		return nil, err
	}
	return cpi.NewSourceAccess(c, r.Access, r.SourceMeta), nil
}

func (c *componentVersionAccessView) GetSourceIndex(id metav1.Identity) int {
	return c.GetDescriptor().GetSourceIndexByIdentity(id)
}

func (c *componentVersionAccessView) GetSourceByIndex(i int) (cpi.SourceAccess, error) {
	if i < 0 || i >= len(c.GetDescriptor().Sources) {
		return nil, errors.ErrInvalid("source index", strconv.Itoa(i))
	}
	r := c.GetDescriptor().Sources[i]
	return cpi.NewSourceAccess(c, r.Access, r.SourceMeta), nil
}

func (c *componentVersionAccessView) GetSources() []cpi.SourceAccess {
	result := []cpi.SourceAccess{}
	for _, r := range c.GetDescriptor().Sources {
		result = append(result, cpi.NewSourceAccess(c, r.Access, r.SourceMeta))
	}
	return result
}

func (c *componentVersionAccessView) GetReferences() compdesc.References {
	return c.GetDescriptor().References
}

func (c *componentVersionAccessView) GetReference(id metav1.Identity) (cpi.ComponentReference, error) {
	return c.GetDescriptor().GetReferenceByIdentity(id)
}

func (c *componentVersionAccessView) GetReferenceIndex(id metav1.Identity) int {
	return c.GetDescriptor().GetReferenceIndexByIdentity(id)
}

func (c *componentVersionAccessView) GetReferenceByIndex(i int) (cpi.ComponentReference, error) {
	if i < 0 || i > len(c.GetDescriptor().References) {
		return cpi.ComponentReference{}, errors.ErrInvalid("reference index", strconv.Itoa(i))
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
