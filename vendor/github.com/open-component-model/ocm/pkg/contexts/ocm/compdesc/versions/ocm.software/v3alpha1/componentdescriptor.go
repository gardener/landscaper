// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v3alpha1

import (
	"errors"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/runtime"
)

var ErrNotFound = errors.New("NotFound")

// ComponentDescriptor defines a versioned component with a source and dependencies.
type ComponentDescriptor struct {
	// TypeMeta specifies the schema version of the component.
	metav1.TypeMeta `json:",inline"`
	// Spec contains the specification of the component.
	metav1.ObjectMeta `json:"metadata"`

	// RepositoryContexts defines the previous repositories of the component
	RepositoryContexts runtime.UnstructuredTypedObjectList `json:"repositoryContexts"`

	Spec ComponentVersionSpec `json:"spec"`
	// Signatures contains a list of signatures for the ComponentDescriptor
	Signatures metav1.Signatures `json:"signatures,omitempty"`
	// NestedDigests described digest information of resources in aggregated
	// omponent versions.
	NestedDigests metav1.NestedDigests `json:"nestedDigests,omitempty"`
}

var _ compdesc.ComponentDescriptorVersion = (*ComponentDescriptor)(nil)

// SchemeVersion returns the actual scheme version of this component descriptor
// representation.
func (cd *ComponentDescriptor) SchemaVersion() string {
	if cd.APIVersion == "" {
		return GroupVersion
	}
	return cd.APIVersion
}

func (cd *ComponentDescriptor) GetName() string {
	return cd.Name
}

// ComponentVersionSpec defines a virtual component with
// a repository context, source and dependencies.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ComponentVersionSpec struct {
	// Sources defines sources that produced the component
	Sources Sources `json:"sources,omitempty"`
	// References references component version dependencies that can be resolved in the current context.
	References References `json:"references,omitempty"`
	// Resources defines all resources that are created by the component and by a third party.
	Resources Resources `json:"resources,omitempty"`
}

const (
	SystemIdentityName    = metav1.SystemIdentityName
	SystemIdentityVersion = metav1.SystemIdentityVersion
)

// ElementMetaAccessor provides generic access an elements meta information.
type ElementMetaAccessor interface {
	GetMeta() *ElementMeta
}

// ElementAccessor provides generic access to list of elements.
type ElementAccessor interface {
	Len() int
	Get(i int) ElementMetaAccessor
}

// ElementMeta defines a object that is uniquely identified by its identity.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ElementMeta struct {
	// Name is the context unique name of the object.
	Name string `json:"name"`
	// Version is the semver version of the object.
	Version string `json:"version"`
	// ExtraIdentity is the identity of an object.
	// An additional label with key "name" ist not allowed
	ExtraIdentity metav1.Identity `json:"extraIdentity,omitempty"`
	// Labels defines an optional set of additional labels
	// describing the object.
	// +optional
	Labels metav1.Labels `json:"labels,omitempty"`
}

// GetName returns the name of the object.
func (o ElementMeta) GetName() string {
	return o.Name
}

// SetName sets the name of the object.
func (o *ElementMeta) SetName(name string) {
	o.Name = name
}

// GetVersion returns the version of the object.
func (o ElementMeta) GetVersion() string {
	return o.Version
}

// SetVersion sets the version of the object.
func (o *ElementMeta) SetVersion(version string) {
	o.Version = version
}

// GetLabels returns the label of the object.
func (o ElementMeta) GetLabels() metav1.Labels {
	return o.Labels
}

// SetLabels sets the labels of the object.
func (o *ElementMeta) SetLabels(labels []metav1.Label) {
	o.Labels = labels
}

// SetExtraIdentity sets the identity of the object.
func (o *ElementMeta) SetExtraIdentity(identity metav1.Identity) {
	o.ExtraIdentity = identity
}

// GetIdentity returns the identity of the object.
func (o *ElementMeta) GetIdentity(accessor ElementAccessor) metav1.Identity {
	identity := o.ExtraIdentity.Copy()
	if identity == nil {
		identity = metav1.Identity{}
	}
	identity[SystemIdentityName] = o.Name
	if accessor != nil {
		found := false
		l := accessor.Len()
		for i := 0; i < l; i++ {
			m := accessor.Get(i).GetMeta()
			if m.Name == o.Name && m.ExtraIdentity.Equals(o.ExtraIdentity) {
				if found {
					identity[SystemIdentityVersion] = o.Version
					break
				}
				found = true
			}
		}
	}
	return identity
}

// GetIdentityDigest returns the digest of the object's identity.
func (o *ElementMeta) GetIdentityDigest(accessor ElementAccessor) []byte {
	return o.GetIdentity(accessor).Digest()
}

// Sources describes a set of source specifications.
type Sources []Source

func (r Sources) Len() int {
	return len(r)
}

func (r Sources) Get(i int) ElementMetaAccessor {
	return &r[i]
}

// Source is the definition of a component's source.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Source struct {
	SourceMeta `json:",inline"`

	Access *runtime.UnstructuredTypedObject `json:"access"`
}

func (s *Source) GetMeta() *ElementMeta {
	return &s.ElementMeta
}

// SourceMeta is the definition of the metadata of a source.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type SourceMeta struct {
	ElementMeta `json:",inline"`
	// Type describes the type of the object.
	Type string `json:"type"`
}

// GetType returns the type of the object.
func (o SourceMeta) GetType() string {
	return o.Type
}

// SetType sets the type of the object.
func (o *SourceMeta) SetType(ttype string) {
	o.Type = ttype
}

// SourceRef defines a reference to a source
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type SourceRef struct {
	// IdentitySelector defines the identity that is used to match a source.
	IdentitySelector metav1.StringMap `json:"identitySelector,omitempty"`
	// Labels defines an optional set of additional labels
	// describing the object.
	// +optional
	Labels metav1.Labels `json:"labels,omitempty"`
}

// Resources describes a set of resource specifications.
type Resources []Resource

func (r Resources) Len() int {
	return len(r)
}

func (r Resources) Get(i int) ElementMetaAccessor {
	return &r[i]
}

// Resource describes a resource dependency of a component.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Resource struct {
	ElementMeta `json:",inline"`

	// Type describes the type of the object.
	Type string `json:"type"`

	// Relation describes the relation of the resource to the component.
	// Can be a local or external resource
	Relation metav1.ResourceRelation `json:"relation,omitempty"`

	// SourceRef defines a list of source names.
	// These names reference the sources defines in `component.sources`.
	SourceRef []SourceRef `json:"srcRef,omitempty"`

	// Access describes the type specific method to
	// access the defined resource.
	Access *runtime.UnstructuredTypedObject `json:"access"`

	// Digest is the optional digest of the referenced resource.
	// +optional
	Digest *metav1.DigestSpec `json:"digest,omitempty"`
}

func (r *Resource) GetMeta() *ElementMeta {
	return &r.ElementMeta
}

// GetType returns the type of the object.
func (r Resource) GetType() string {
	return r.Type
}

// SetType sets the type of the object.
func (r *Resource) SetType(ttype string) {
	r.Type = ttype
}

type References []Reference

func (r References) Len() int {
	return len(r)
}

func (r References) Get(i int) ElementMetaAccessor {
	return &r[i]
}

// Reference describes the reference to another component in the registry.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Reference struct {
	ElementMeta `json:",inline"`
	// ComponentName describes the remote name of the referenced object
	ComponentName string `json:"componentName"`
	// Digest is the optional digest of the referenced component.
	// +optional
	Digest *metav1.DigestSpec `json:"digest,omitempty"`
}

func (r *Reference) GetMeta() *ElementMeta {
	return &r.ElementMeta
}
