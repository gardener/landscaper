// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"bytes"
	"fmt"

	v1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils/selector"
)

// GetEffectiveRepositoryContext returns the currently active repository context.
func (cd *ComponentDescriptor) GetEffectiveRepositoryContext() *runtime.UnstructuredTypedObject {
	if len(cd.RepositoryContexts) == 0 {
		return nil
	}
	return cd.RepositoryContexts[len(cd.RepositoryContexts)-1]
}

// AddRepositoryContext appends the given repository context to components descriptor repository history.
// The context is not appended if the effective repository context already matches the current context.
func (cd *ComponentDescriptor) AddRepositoryContext(repoCtx runtime.TypedObject) error {
	effective, err := runtime.ToUnstructuredTypedObject(cd.GetEffectiveRepositoryContext())
	if err != nil {
		return err
	}
	uRepoCtx, err := runtime.ToUnstructuredTypedObject(repoCtx)
	if err != nil {
		return err
	}
	if !runtime.UnstructuredTypesEqual(effective, uRepoCtx) {
		cd.RepositoryContexts = append(cd.RepositoryContexts, uRepoCtx)
	}
	return nil
}

// GetComponentReferences returns all component references that matches the given selectors.
func (cd *ComponentDescriptor) GetComponentReferences(selectors ...IdentitySelector) ([]ComponentReference, error) {
	refs := make([]ComponentReference, 0)
	for _, ref := range cd.References {
		ok, err := selector.MatchSelectors(ref.GetIdentity(cd.References), selectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", ref.Name, err)
		}
		if ok {
			refs = append(refs, ref)
		}
	}
	if len(refs) == 0 {
		return refs, NotFound
	}
	return refs, nil
}

// GetResourceByIdentity returns resource that matches the given identity.
func (cd *ComponentDescriptor) GetResourceByIdentity(id v1.Identity) (Resource, error) {
	dig := id.Digest()
	for _, res := range cd.Resources {
		if bytes.Equal(res.GetIdentityDigest(cd.Resources), dig) {
			return res, nil
		}
	}
	return Resource{}, NotFound
}

// GetResourceAccessByIdentity returns a pointer to the resource that matches the given identity.
func (cd *ComponentDescriptor) GetResourceAccessByIdentity(id v1.Identity) *Resource {
	dig := id.Digest()
	for i, res := range cd.Resources {
		if bytes.Equal(res.GetIdentityDigest(cd.Resources), dig) {
			return &cd.Resources[i]
		}
	}
	return nil
}

// GetResourceIndexByIdentity returns the index of the resource that matches the given identity.
func (cd *ComponentDescriptor) GetResourceIndexByIdentity(id v1.Identity) int {
	dig := id.Digest()
	for i, res := range cd.Resources {
		if bytes.Equal(res.GetIdentityDigest(cd.Resources), dig) {
			return i
		}
	}
	return -1
}

// GetResourceByJSONScheme returns resources that match the given selectors.
func (cd *ComponentDescriptor) GetResourceByJSONScheme(src interface{}) (Resources, error) {
	sel, err := selector.NewJSONSchemaSelectorFromGoStruct(src)
	if err != nil {
		return nil, err
	}
	return cd.GetResourcesByIdentitySelectors(sel)
}

// GetResourceByDefaultSelector returns resources that match the given selectors.
func (cd *ComponentDescriptor) GetResourceByDefaultSelector(sel interface{}) (Resources, error) {
	identitySelector, err := selector.ParseDefaultSelector(sel)
	if err != nil {
		return nil, fmt.Errorf("unable to parse selector: %w", err)
	}
	return cd.GetResourcesByIdentitySelectors(identitySelector)
}

// GetResourceByRegexSelector returns resources that match the given selectors.
func (cd *ComponentDescriptor) GetResourceByRegexSelector(sel interface{}) (Resources, error) {
	identitySelector, err := selector.ParseRegexSelector(sel)
	if err != nil {
		return nil, fmt.Errorf("unable to parse selector: %w", err)
	}
	return cd.GetResourcesByIdentitySelectors(identitySelector)
}

// GetResourcesByIdentitySelectors returns resources that match the given identity selectors.
func (cd *ComponentDescriptor) GetResourcesByIdentitySelectors(selectors ...IdentitySelector) (Resources, error) {
	return cd.GetResourcesBySelectors(selectors, nil)
}

// GetResourcesByResourceSelectors returns resources that match the given resource selectors.
func (cd *ComponentDescriptor) GetResourcesByResourceSelectors(selectors ...ResourceSelector) (Resources, error) {
	return cd.GetResourcesBySelectors(nil, selectors)
}

// GetResourcesBySelectors returns resources that match the given selector.
func (cd *ComponentDescriptor) GetResourcesBySelectors(selectors []IdentitySelector, resourceSelectors []ResourceSelector) (Resources, error) {
	resources := make(Resources, 0)
	for i := range cd.Resources {
		selctx := NewResourceSelectionContext(i, cd.Resources)
		if len(selectors) > 0 {
			ok, err := selector.MatchSelectors(selctx.Identity(), selectors...)
			if err != nil {
				return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
			}
			if !ok {
				continue
			}
		}
		ok, err := MatchResourceByResourceSelector(selctx, resourceSelectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
		}
		if !ok {
			continue
		}
		resources = append(resources, *selctx.Resource)
	}
	if len(resources) == 0 {
		return resources, NotFound
	}
	return resources, nil
}

// GetExternalResources returns external resource with the given type, name and version.
func (cd *ComponentDescriptor) GetExternalResources(rtype, name, version string) (Resources, error) {
	return cd.GetResourcesBySelectors(
		[]selector.Interface{
			ByName(name),
			ByVersion(version),
		},
		[]ResourceSelector{
			ByResourceType(rtype),
			ByRelation(v1.ExternalRelation),
		})
}

// GetExternalResource returns external resource with the given type, name and version.
// If multiple resources match, the first one is returned.
func (cd *ComponentDescriptor) GetExternalResource(rtype, name, version string) (Resource, error) {
	resources, err := cd.GetExternalResources(rtype, name, version)
	if err != nil {
		return Resource{}, err
	}
	// at least one resource must be defined, otherwise the getResourceBySelectors functions returns a NotFound err.
	return resources[0], nil
}

// GetLocalResources returns all local resources with the given type, name and version.
func (cd *ComponentDescriptor) GetLocalResources(rtype, name, version string) (Resources, error) {
	return cd.GetResourcesBySelectors(
		[]selector.Interface{
			ByName(name),
			ByVersion(version),
		},
		[]ResourceSelector{
			ByResourceType(rtype),
			ByRelation(v1.LocalRelation),
		})
}

// GetLocalResource returns a local resource with the given type, name and version.
// If multiple resources match, the first one is returned.
func (cd *ComponentDescriptor) GetLocalResource(rtype, name, version string) (Resource, error) {
	resources, err := cd.GetLocalResources(rtype, name, version)
	if err != nil {
		return Resource{}, err
	}
	// at least one resource must be defined, otherwise the getResourceBySelectors functions returns a NotFound err.
	return resources[0], nil
}

// GetResourcesByType returns all resources that match the given type and selectors.
func (cd *ComponentDescriptor) GetResourcesByType(rtype string, selectors ...IdentitySelector) (Resources, error) {
	return cd.GetResourcesBySelectors(
		selectors,
		[]ResourceSelector{
			ByResourceType(rtype),
		})
}

// GetResourcesByName returns all local and external resources with a name.
func (cd *ComponentDescriptor) GetResourcesByName(name string, selectors ...IdentitySelector) (Resources, error) {
	return cd.GetResourcesBySelectors(
		generics.AppendedSlice[IdentitySelector](selectors, ByName(name)),
		nil)
}

// GetResourceIndex returns the index of a given resource.
// If the index is not found -1 is returned.
func (cd *ComponentDescriptor) GetResourceIndex(res *ResourceMeta) int {
	id := res.GetIdentity(cd.Resources)
	for i, cur := range cd.Resources {
		if cur.GetIdentity(cd.Resources).Equals(id) {
			return i
		}
	}
	return -1
}

// GetComponentReferenceIndex returns the index of a given component reference.
// If the index is not found -1 is returned.
func (cd *ComponentDescriptor) GetComponentReferenceIndex(ref ComponentReference) int {
	id := ref.GetIdentityDigest(cd.References)
	for i, cur := range cd.References {
		if bytes.Equal(cur.GetIdentityDigest(cd.References), id) {
			return i
		}
	}
	return -1
}

// GetSourceByIdentity returns source that match the given identity.
func (cd *ComponentDescriptor) GetSourceByIdentity(id v1.Identity) (Source, error) {
	dig := id.Digest()
	for _, res := range cd.Sources {
		if bytes.Equal(res.GetIdentityDigest(cd.Sources), dig) {
			return res, nil
		}
	}
	return Source{}, NotFound
}

// GetSourceByIdentity returns a pointer to the source that matches the given identity.
func (cd *ComponentDescriptor) GetSourceAccessByIdentity(id v1.Identity) *Source {
	dig := id.Digest()
	for i, res := range cd.Sources {
		if bytes.Equal(res.GetIdentityDigest(cd.Sources), dig) {
			return &cd.Sources[i]
		}
	}
	return nil
}

// GetSourceIndexByIdentity returns the index of the source that matches the given identity.
func (cd *ComponentDescriptor) GetSourceIndexByIdentity(id v1.Identity) int {
	dig := id.Digest()
	for i, res := range cd.Sources {
		if bytes.Equal(res.GetIdentityDigest(cd.Sources), dig) {
			return i
		}
	}
	return -1
}

// GetSourcesByIdentitySelectors returns references that match the given selector.
func (cd *ComponentDescriptor) GetSourcesByIdentitySelectors(selectors ...IdentitySelector) (Sources, error) {
	srcs := make(Sources, 0)
	for _, src := range cd.Sources {
		ok, err := selector.MatchSelectors(src.GetIdentity(cd.Sources), selectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for source %s: %w", src.Name, err)
		}
		if ok {
			srcs = append(srcs, src)
		}
	}
	if len(srcs) == 0 {
		return srcs, NotFound
	}
	return srcs, nil
}

// GetSourceIndex returns the index of a given source.
// If the index is not found -1 is returned.
func (cd *ComponentDescriptor) GetSourceIndex(src *SourceMeta) int {
	id := src.GetIdentityDigest(cd.Sources)
	for i, cur := range cd.Sources {
		if bytes.Equal(cur.GetIdentityDigest(cd.Sources), id) {
			return i
		}
	}
	return -1
}

// GetReferenceByIdentity returns reference that matches the given identity.
func (cd *ComponentDescriptor) GetReferenceByIdentity(id v1.Identity) (ComponentReference, error) {
	dig := id.Digest()
	for _, ref := range cd.References {
		if bytes.Equal(ref.GetIdentityDigest(cd.Resources), dig) {
			return ref, nil
		}
	}
	return ComponentReference{}, errors.ErrNotFound(KIND_REFERENCE, id.String())
}

// GetReferenceAccessByIdentity returns a pointer to the reference that matches the given identity.
func (cd *ComponentDescriptor) GetReferenceAccessByIdentity(id v1.Identity) *ComponentReference {
	dig := id.Digest()
	for i, ref := range cd.References {
		if bytes.Equal(ref.GetIdentityDigest(cd.Resources), dig) {
			return &cd.References[i]
		}
	}
	return nil
}

// GetReferenceIndexByIdentity returns the index of the reference that matches the given identity.
func (cd *ComponentDescriptor) GetReferenceIndexByIdentity(id v1.Identity) int {
	dig := id.Digest()
	for i, ref := range cd.References {
		if bytes.Equal(ref.GetIdentityDigest(cd.Resources), dig) {
			return i
		}
	}
	return -1
}

// GetReferencesByName returns references that match the given name.
func (cd *ComponentDescriptor) GetReferencesByName(name string, selectors ...IdentitySelector) (References, error) {
	return cd.GetReferencesBySelectors(
		generics.AppendedSlice[IdentitySelector](selectors, ByName(name)),
		nil)
}

// GetReferencesByIdentitySelectors returns resources that match the given identity selectors.
func (cd *ComponentDescriptor) GetReferencesByIdentitySelectors(selectors ...IdentitySelector) (References, error) {
	return cd.GetReferencesBySelectors(selectors, nil)
}

// GetReferencesByReferenceSelectors returns resources that match the given resource selectors.
func (cd *ComponentDescriptor) GetReferencesByReferenceSelectors(selectors ...ReferenceSelector) (References, error) {
	return cd.GetReferencesBySelectors(nil, selectors)
}

// GetReferencesBySelectors returns resources that match the given selector.
func (cd *ComponentDescriptor) GetReferencesBySelectors(selectors []IdentitySelector, referenceSelectors []ReferenceSelector) (References, error) {
	references := make(References, 0)
	for i := range cd.References {
		selctx := NewReferenceSelectionContext(i, cd.References)
		if len(selectors) > 0 {
			ok, err := selector.MatchSelectors(selctx.Identity(), selectors...)
			if err != nil {
				return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
			}
			if !ok {
				continue
			}
		}
		ok, err := MatchReferencesByReferenceSelector(selctx, referenceSelectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
		}
		if !ok {
			continue
		}
		references = append(references, *selctx.ComponentReference)
	}
	if len(references) == 0 {
		return references, NotFound
	}
	return references, nil
}

// GetReferenceIndex returns the index of a given source.
// If the index is not found -1 is returned.
func (cd *ComponentDescriptor) GetReferenceIndex(src *ElementMeta) int {
	id := src.GetIdentityDigest(cd.References)
	for i, cur := range cd.References {
		if bytes.Equal(cur.GetIdentityDigest(cd.References), id) {
			return i
		}
	}
	return -1
}

// GetSignatureIndex returns the index of the signature with the given name
// If the index is not found -1 is returned.
func (cd *ComponentDescriptor) GetSignatureIndex(name string) int {
	return cd.Signatures.GetIndex(name)
}
