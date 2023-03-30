// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"bytes"
	"fmt"

	v1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils/selector"
)

type IdentitySelector = selector.Interface

// ResourceSelectorFunc defines a function to filter a resource.
type ResourceSelectorFunc = func(obj Resource) (bool, error)

// MatchResourceSelectorFuncs applies all resource selector against the given resource object.
func MatchResourceSelectorFuncs(obj Resource, resourceSelectors ...ResourceSelectorFunc) (bool, error) {
	for _, sel := range resourceSelectors {
		ok, err := sel(obj)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// NewTypeResourceSelector creates a new resource selector that
// selects a resource based on its type.
func NewTypeResourceSelector(ttype string) ResourceSelectorFunc {
	return func(obj Resource) (bool, error) {
		return obj.GetType() == ttype, nil
	}
}

// NewVersionResourceSelector creates a new resource selector that
// selects a resource based on its version.
func NewVersionResourceSelector(version string) ResourceSelectorFunc {
	return func(obj Resource) (bool, error) {
		return obj.GetVersion() == version, nil
	}
}

// NewRelationResourceSelector creates a new resource selector that
// selects a resource based on its relation type.
func NewRelationResourceSelector(relation v1.ResourceRelation) ResourceSelectorFunc {
	return func(obj Resource) (bool, error) {
		return obj.Relation == relation, nil
	}
}

// NewNameSelector creates a new selector that matches a resource name.
func NewNameSelector(name string) selector.Interface {
	return selector.DefaultSelector{
		SystemIdentityName: name,
	}
}

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

// GetResourceByIdentity returns resource that match the given identity.
func (cd *ComponentDescriptor) GetResourceByIdentity(id v1.Identity) (Resource, error) {
	dig := id.Digest()
	for _, res := range cd.Resources {
		if bytes.Equal(res.GetIdentityDigest(cd.Resources), dig) {
			return res, nil
		}
	}
	return Resource{}, NotFound
}

// GetResourceByJSONScheme returns resources that match the given selectors.
func (cd *ComponentDescriptor) GetResourceByJSONScheme(src interface{}) ([]Resource, error) {
	sel, err := selector.NewJSONSchemaSelectorFromGoStruct(src)
	if err != nil {
		return nil, err
	}
	return cd.GetResourcesBySelector(sel)
}

// GetResourceByDefaultSelector returns resources that match the given selectors.
func (cd *ComponentDescriptor) GetResourceByDefaultSelector(sel interface{}) ([]Resource, error) {
	identitySelector, err := selector.ParseDefaultSelector(sel)
	if err != nil {
		return nil, fmt.Errorf("unable to parse selector: %w", err)
	}
	return cd.GetResourcesBySelector(identitySelector)
}

// GetResourceByRegexSelector returns resources that match the given selectors.
func (cd *ComponentDescriptor) GetResourceByRegexSelector(sel interface{}) ([]Resource, error) {
	identitySelector, err := selector.ParseRegexSelector(sel)
	if err != nil {
		return nil, fmt.Errorf("unable to parse selector: %w", err)
	}
	return cd.GetResourcesBySelector(identitySelector)
}

// GetResourcesBySelector returns resources that match the given selector.
func (cd *ComponentDescriptor) GetResourcesBySelector(selectors ...IdentitySelector) ([]Resource, error) {
	resources := make([]Resource, 0)
	for _, res := range cd.Resources {
		ok, err := selector.MatchSelectors(res.GetIdentity(cd.Resources), selectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", res.Name, err)
		}
		if ok {
			resources = append(resources, res)
		}
	}
	if len(resources) == 0 {
		return resources, NotFound
	}
	return resources, nil
}

// GetResourcesBySelector returns resources that match the given selector.
func (cd *ComponentDescriptor) getResourceBySelectors(selectors []IdentitySelector, resourceSelectors []ResourceSelectorFunc) ([]Resource, error) {
	resources := make([]Resource, 0)
	for _, res := range cd.Resources {
		ok, err := selector.MatchSelectors(res.GetIdentity(cd.Resources), selectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", res.Name, err)
		}
		if !ok {
			continue
		}
		ok, err = MatchResourceSelectorFuncs(res, resourceSelectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", res.Name, err)
		}
		if !ok {
			continue
		}
		resources = append(resources, res)
	}
	if len(resources) == 0 {
		return resources, NotFound
	}
	return resources, nil
}

// GetExternalResources returns external resource with the given type, name and version.
func (cd *ComponentDescriptor) GetExternalResources(rtype, name, version string) ([]Resource, error) {
	return cd.getResourceBySelectors(
		[]selector.Interface{NewNameSelector(name)},
		[]ResourceSelectorFunc{
			NewTypeResourceSelector(rtype),
			NewVersionResourceSelector(version),
			NewRelationResourceSelector(v1.ExternalRelation),
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
func (cd *ComponentDescriptor) GetLocalResources(rtype, name, version string) ([]Resource, error) {
	return cd.getResourceBySelectors(
		[]selector.Interface{NewNameSelector(name)},
		[]ResourceSelectorFunc{
			NewTypeResourceSelector(rtype),
			NewVersionResourceSelector(version),
			NewRelationResourceSelector(v1.LocalRelation),
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
func (cd *ComponentDescriptor) GetResourcesByType(rtype string, selectors ...IdentitySelector) ([]Resource, error) {
	return cd.getResourceBySelectors(
		selectors,
		[]ResourceSelectorFunc{
			NewTypeResourceSelector(rtype),
		})
}

// GetResourcesByName returns all local and external resources with a name.
func (cd *ComponentDescriptor) GetResourcesByName(name string, selectors ...IdentitySelector) ([]Resource, error) {
	return cd.getResourceBySelectors(
		append(selectors, NewNameSelector(name)),
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

// GetComponentReferenceByIdentity returns reference that match the given identity.
func (cd *ComponentDescriptor) GetComponentReferenceByIdentity(id v1.Identity) (ComponentReference, error) {
	dig := id.Digest()
	for _, ref := range cd.References {
		if bytes.Equal(ref.GetIdentityDigest(cd.References), dig) {
			return ref, nil
		}
	}
	return ComponentReference{}, NotFound
}

// GetComponentReferencesByName returns references that match the given name.
func (cd *ComponentDescriptor) GetComponentReferencesByName(name string) []ComponentReference {
	var refs []ComponentReference
	for _, ref := range cd.References {
		if ref.Name == name {
			refs = append(refs, ref)
		}
	}
	return refs
}

// GetSourceByIdentity returns source that match the given identity.
func (cd *ComponentDescriptor) GetSourceByIdentity(id v1.Identity) (Source, error) {
	dig := id.Digest()
	for _, res := range cd.Sources {
		if bytes.Equal(res.GetIdentityDigest(cd.Resources), dig) {
			return res, nil
		}
	}
	return Source{}, NotFound
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

// GetReferenceByIdentity returns reference that match the given identity.
func (cd *ComponentDescriptor) GetReferenceByIdentity(id v1.Identity) (ComponentReference, error) {
	dig := id.Digest()
	for _, ref := range cd.References {
		if bytes.Equal(ref.GetIdentityDigest(cd.Resources), dig) {
			return ref, nil
		}
	}
	return ComponentReference{}, errors.ErrNotFound(KIND_REFERENCE, id.String())
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
	for i, cur := range cd.Signatures {
		if cur.Name == name {
			return i
		}
	}
	return -1
}
