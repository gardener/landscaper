// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2

import (
	"encoding/json"
	"errors"
)

const SchemaVersion = "v2"

var (
	NotFound = errors.New("NotFound")
)

// Metadata defines the metadata of the component descriptor.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Metadata struct {
	// Version is the schema version of the component descriptor.
	Version string `json:"schemaVersion"`
}

// ProviderType describes the provider type of component in the origin's context.
// Defines whether the component is created by a third party or internally.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ProviderType string

const (
	// InternalProvider defines a internal provider type
	// which describes a internally maintained component in the origin's context.
	InternalProvider ProviderType = "internal"
	// ExternalProvider defines a external provider type
	// which describes a component maintained by a third party vendor in the origin's context.
	ExternalProvider ProviderType = "external"
)

// ResourceRelation describes the type of a resource.
// Defines whether the component is created by a third party or internally.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ResourceRelation string

const (
	// LocalRelation defines a internal relation
	// which describes a internally maintained resource in the origin's context.
	LocalRelation ResourceRelation = "local"
	// ExternalRelation defines a external relation
	// which describes a resource maintained by a third party vendor in the origin's context.
	ExternalRelation ResourceRelation = "external"
)

// ComponentDescriptor defines a versioned component with a source and dependencies.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ComponentDescriptor struct {
	// Metadata specifies the schema version of the component.
	Metadata Metadata `json:"meta"`
	// Spec contains the specification of the component.
	ComponentSpec `json:"component"`

	// Signatures contains a list of signatures for the ComponentDescriptor
	Signatures []Signature `json:"signatures,omitempty"`
}

// ComponentSpec defines a virtual component with
// a repository context, source and dependencies.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ComponentSpec struct {
	ObjectMeta `json:",inline"`
	// RepositoryContexts defines the previous repositories of the component
	RepositoryContexts []*UnstructuredTypedObject `json:"repositoryContexts"`
	// Provider defines the provider type of a component.
	// It can be external or internal.
	Provider ProviderType `json:"provider"`
	// Sources defines sources that produced the component
	Sources []Source `json:"sources"`
	// ComponentReferences references component dependencies that can be resolved in the current context.
	ComponentReferences []ComponentReference `json:"componentReferences"`
	// Resources defines all resources that are created by the component and by a third party.
	Resources []Resource `json:"resources"`
}

// ObjectMeta defines a object that is uniquely identified by its name and version.
// +k8s:deepcopy-gen=true
type ObjectMeta struct {
	// Name is the context unique name of the object.
	Name string `json:"name"`
	// Version is the semver version of the object.
	Version string `json:"version"`
	// Labels defines an optional set of additional labels
	// describing the object.
	// +optional
	Labels Labels `json:"labels,omitempty"`
}

// GetName returns the name of the object.
func (o ObjectMeta) GetName() string {
	return o.Name
}

// SetName sets the name of the object.
func (o *ObjectMeta) SetName(name string) {
	o.Name = name
}

// GetVersion returns the version of the object.
func (o ObjectMeta) GetVersion() string {
	return o.Version
}

// SetVersion sets the version of the object.
func (o *ObjectMeta) SetVersion(version string) {
	o.Version = version
}

// GetLabels returns the label of the object.
func (o ObjectMeta) GetLabels() Labels {
	return o.Labels
}

// SetLabels sets the labels of the object.
func (o *ObjectMeta) SetLabels(labels []Label) {
	o.Labels = labels
}

const (
	SystemIdentityName    = "name"
	SystemIdentityVersion = "version"
)

// Identity describes the identity of an object.
// Only ascii characters are allowed
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Identity map[string]string

// Digest returns the object digest of an identity
func (i Identity) Digest() []byte {
	data, _ := json.Marshal(i)
	return data
}

// Match implements the selector interface.
func (i Identity) Match(obj map[string]string) (bool, error) {
	for k, v := range i {
		if obj[k] != v {
			return false, nil
		}
	}
	return true, nil
}

// IdentityObjectMeta defines a object that is uniquely identified by its identity.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type IdentityObjectMeta struct {
	// Name is the context unique name of the object.
	Name string `json:"name"`
	// Version is the semver version of the object.
	Version string `json:"version"`
	// Type describes the type of the object.
	Type string `json:"type"`
	// ExtraIdentity is the identity of an object.
	// An additional label with key "name" ist not allowed
	ExtraIdentity Identity `json:"extraIdentity,omitempty"`
	// Labels defines an optional set of additional labels
	// describing the object.
	// +optional
	Labels Labels `json:"labels,omitempty"`
}

// GetName returns the name of the object.
func (o IdentityObjectMeta) GetName() string {
	return o.Name
}

// SetName sets the name of the object.
func (o *IdentityObjectMeta) SetName(name string) {
	o.Name = name
}

// GetVersion returns the version of the object.
func (o IdentityObjectMeta) GetVersion() string {
	return o.Version
}

// SetVersion sets the version of the object.
func (o *IdentityObjectMeta) SetVersion(version string) {
	o.Version = version
}

// GetType returns the type of the object.
func (o IdentityObjectMeta) GetType() string {
	return o.Type
}

// SetType sets the type of the object.
func (o *IdentityObjectMeta) SetType(ttype string) {
	o.Type = ttype
}

// GetLabels returns the label of the object.
func (o IdentityObjectMeta) GetLabels() Labels {
	return o.Labels
}

// SetLabels sets the labels of the object.
func (o *IdentityObjectMeta) SetLabels(labels []Label) {
	o.Labels = labels
}

// SetExtraIdentity sets the identity of the object.
func (o *IdentityObjectMeta) SetExtraIdentity(identity Identity) {
	o.ExtraIdentity = identity
}

// GetIdentity returns the identity of the object.
func (o *IdentityObjectMeta) GetIdentity() Identity {
	identity := map[string]string{}
	for k, v := range o.ExtraIdentity {
		identity[k] = v
	}
	identity[SystemIdentityName] = o.Name
	return identity
}

// GetIdentityDigest returns the digest of the object's identity.
func (o *IdentityObjectMeta) GetIdentityDigest() []byte {
	return o.GetIdentity().Digest()
}

// ObjectType describes the type of a object
// +k8s:deepcopy-gen=true
type ObjectType struct {
	// Type describes the type of the object.
	Type string `json:"type"`
}

// GetType returns the type of the object.
func (t ObjectType) GetType() string {
	return t.Type
}

// SetType sets the type of the object.
func (t *ObjectType) SetType(ttype string) {
	t.Type = ttype
}

// Label is a label that can be set on objects.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Label struct {
	// Name is the unique name of the label.
	Name string `json:"name"`
	// Value is the json/yaml data of the label
	Value json.RawMessage `json:"value"`
}

// Labels describe a list of labels
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Labels []Label

// Get returns the label witht the given name
func (l Labels) Get(name string) ([]byte, bool) {
	for _, label := range l {
		if label.Name == name {
			return label.Value, true
		}
	}
	return nil, false
}

// NameAccessor describes a accessor for a named object.
type NameAccessor interface {
	// GetName returns the name of the object.
	GetName() string
	// SetName sets the name of the object.
	SetName(name string)
}

// VersionAccessor describes a accessor for a versioned object.
type VersionAccessor interface {
	// GetVersion returns the version of the object.
	GetVersion() string
	// SetVersion sets the version of the object.
	SetVersion(version string)
}

// LabelsAccessor describes a accessor for a labeled object.
type LabelsAccessor interface {
	// GetLabels returns the labels of the object.
	GetLabels() Labels
	// SetLabels sets the labels of the object.
	SetLabels(labels []Label)
}

// ObjectMetaAccessor describes a accessor for named and versioned object.
type ObjectMetaAccessor interface {
	NameAccessor
	VersionAccessor
	LabelsAccessor
}

// TypedObjectAccessor defines the accessor for a typed component with additional data.
type TypedObjectAccessor interface {
	// GetType returns the type of the access object.
	GetType() string
	// SetType sets the type of the access object.
	SetType(ttype string)
}

// Repository is a specific type that indicated a typed repository object.
type Repository TypedObjectAccessor

// Source is the definition of a component's source.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Source struct {
	IdentityObjectMeta `json:",inline"`
	Access             *UnstructuredTypedObject `json:"access"`
}

// SourceRef defines a reference to a source
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type SourceRef struct {
	// IdentitySelector defines the identity that is used to match a source.
	IdentitySelector map[string]string `json:"identitySelector,omitempty"`
	// Labels defines an optional set of additional labels
	// describing the object.
	// +optional
	Labels Labels `json:"labels,omitempty"`
}

// Resource describes a resource dependency of a component.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Resource struct {
	IdentityObjectMeta `json:",inline"`

	// Digest is the optional digest of the referenced resource.
	// +optional
	Digest *DigestSpec `json:"digest,omitempty"`

	// Relation describes the relation of the resource to the component.
	// Can be a local or external resource
	Relation ResourceRelation `json:"relation,omitempty"`

	// SourceRef defines a list of source names.
	// These names reference the sources defines in `component.sources`.
	SourceRef []SourceRef `json:"srcRef,omitempty"`

	// Access describes the type specific method to
	// access the defined resource.
	Access *UnstructuredTypedObject `json:"access"`
}

// ComponentReference describes the reference to another component in the registry.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ComponentReference struct {
	// Name is the context unique name of the object.
	Name string `json:"name"`
	// ComponentName describes the remote name of the referenced object
	ComponentName string `json:"componentName"`
	// Version is the semver version of the object.
	Version string `json:"version"`
	// ExtraIdentity is the identity of an object.
	// An additional label with key "name" ist not allowed
	ExtraIdentity Identity `json:"extraIdentity,omitempty"`
	// Digest is the optional digest of the referenced component.
	// +optional
	Digest *DigestSpec `json:"digest,omitempty"`
	// Labels defines an optional set of additional labels
	// describing the object.
	// +optional
	Labels Labels `json:"labels,omitempty"`
}

// GetName returns the name of the object.
func (o ComponentReference) GetName() string {
	return o.Name
}

// SetName sets the name of the object.
func (o *ComponentReference) SetName(name string) {
	o.Name = name
}

// GetVersion returns the version of the object.
func (o ComponentReference) GetVersion() string {
	return o.Version
}

// SetVersion sets the version of the object.
func (o *ComponentReference) SetVersion(version string) {
	o.Version = version
}

// GetLabels returns the label of the object.
func (o ComponentReference) GetLabels() Labels {
	return o.Labels
}

// SetLabels sets the labels of the object.
func (o *ComponentReference) SetLabels(labels []Label) {
	o.Labels = labels
}

// GetIdentity returns the identity of the object.
func (o *ComponentReference) GetIdentity() Identity {
	identity := map[string]string{}
	for k, v := range o.ExtraIdentity {
		identity[k] = v
	}
	identity[SystemIdentityName] = o.Name
	return identity
}

// GetIdentityDigest returns the digest of the object's identity.
func (o *ComponentReference) GetIdentityDigest() []byte {
	return o.GetIdentity().Digest()
}

// DigestSpec defines a digest.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type DigestSpec struct {
	HashAlgorithm          string `json:"hashAlgorithm"`
	NormalisationAlgorithm string `json:"normalisationAlgorithm"`
	Value                  string `json:"value"`
}

// SignatureSpec defines a signature.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type SignatureSpec struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
	MediaType string `json:"mediaType"`
}

const (
	// SignaturePEMBlockType defines the type of a signature pem block.
	SignaturePEMBlockType = "SIGNATURE"

	// SignatureAlgorithmHeader defines a pem header where the signature algorithm is defined.
	SignatureAlgorithmHeader = "Signature Algorithm"

	// MediaTypePEM defines the media type for pem formatted data.
	MediaTypePEM = "application/x-pem-file"

	// MediaTypeRSASignature defines the media type for a plain RSA signature.
	MediaTypeRSASignature = "application/vnd.ocm.signature.rsa"

	// RSAPKCS1v15 defines the type for the RSA PKCS #1 v1.5 signature algorithm
	RSAPKCS1v15 = "RSASSA-PKCS1-V1_5"

	// ExcludeFromSignature used in digest field for normalisationAlgorithm (in combination with NoDigest for hashAlgorithm and value)
	// to indicate the resource content should not be part of the signature
	ExcludeFromSignature = "EXCLUDE-FROM-SIGNATURE"

	// NoDigest used in digest field for hashAlgorithm and value (in combination with ExcludeFromSignature for normalisationAlgorithm)
	// to indicate the resource content should not be part of the signature
	NoDigest = "NO-DIGEST"
)

//NewExcludeFromSignatureDigest returns the special digest notation to indicate the resource content should not be part of the signature
func NewExcludeFromSignatureDigest() *DigestSpec {
	return &DigestSpec{
		HashAlgorithm:          NoDigest,
		NormalisationAlgorithm: ExcludeFromSignature,
		Value:                  NoDigest,
	}
}

// NormalisationAlgorithm types and versions the algorithm used for digest generation.
type NormalisationAlgorithm string

const (
	JsonNormalisationV1 NormalisationAlgorithm = "jsonNormalisation/v1"
	OciArtifactDigestV1 NormalisationAlgorithm = "ociArtifactDigest/v1"
	GenericBlobDigestV1 NormalisationAlgorithm = "genericBlobDigest/v1"
)

// Signature defines a digest and corresponding signature, identifyable by name.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Signature struct {
	Name      string        `json:"name"`
	Digest    DigestSpec    `json:"digest"`
	Signature SignatureSpec `json:"signature"`
}
