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
type Metadata struct {
	// Version is the schema version of the component descriptor.
	Version string `json:"schemaVersion"`
}

// ProviderType describes the provider type of component in the origin's context.
// Defines whether the component is created by a third party or internally.
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
type ResourceRelation string

const (
	// LocalRelation defines a internal relation
	// which describes a internally maintained resource in the origin's context.
	LocalRelation ResourceRelation = "local"
	// ExternalRelation defines a external relation
	// which describes a resource maintained by a third party vendor in the origin's context.
	ExternalRelation ResourceRelation = "external"
)

// Spec defines a versioned virtual component with a source and dependencies.
type ComponentDescriptor struct {
	// Metadata specifies the schema version of the component.
	Metadata Metadata `json:"meta"`
	// Spec contains the specification of the component.
	ComponentSpec `json:"component"`
}

// ComponentSpec defines a virtual component with
// a repository context, source and dependencies.
type ComponentSpec struct {
	ObjectMeta `json:",inline"`
	// RepositoryContexts defines the previous repositories of the component
	RepositoryContexts []RepositoryContext `json:"repositoryContexts"`
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

// RepositoryContext describes a repository context.
type RepositoryContext struct {
	// Type defines the type of the component repository to resolve references.
	Type string `json:"type"`
	// BaseURL is the base url of the repository to resolve components.
	BaseURL string `json:"baseUrl"`
}

// ObjectMeta defines a object that is uniquely identified by its name and version.
type ObjectMeta struct {
	// Name is the context unique name of the object.
	Name string `json:"name"`
	// Version is the semver version of the object.
	Version string `json:"version"`
	// Labels defines an optional set of additional labels
	// describing the object.
	// +optional
	Labels []Label `json:"labels,omitempty"`
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
func (o ObjectMeta) GetLabels() []Label {
	return o.Labels
}

// SetLabels sets the labels of the object.
func (o *ObjectMeta) SetLabels(labels []Label) {
	o.Labels = labels
}

// ObjectType describes the type of a object
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
type Label struct {
	// Name is the unique name of the label.
	Name string `json:"name"`
	// Value is the json/yaml data of the label
	Value json.RawMessage `json:"value"`
}

// ComponentReference describes the reference to another component in the registry.	// Source is the definition of a component's source.
type ComponentReference struct {
	// Name is the context unique name of the object.
	Name string `json:"name"`
	// ComponentName describes the remote name of the referenced object
	ComponentName string `json:"componentName"`
	// Version is the semver version of the object.
	Version string `json:"version"`
	// Labels defines an optional set of additional labels
	// describing the object.
	// +optional
	Labels []Label `json:"labels,omitempty"`
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
func (o ComponentReference) GetLabels() []Label {
	return o.Labels
}

// SetLabels sets the labels of the object.
func (o *ComponentReference) SetLabels(labels []Label) {
	o.Labels = labels
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
	GetLabels() []Label
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
	// GetData returns the custom data of a component.
	GetData() ([]byte, error)
	// SetData sets the custom data of a component.
	SetData([]byte) error
}

// Source is the definition of a component's source.
type Source struct {
	Name                string `json:"name"`
	TypedObjectAccessor `json:",inline"`
	Access              TypedObjectAccessor `json:"access"`
}

// GetName returns the name of the source.
func (s Source) GetName() string {
	return s.Name
}

// SetName sets the name of the source.
func (s *Source) SetName(name string) {
	s.Name = name
}

// jsonSource is the internal representation of a Source
// that is used to marshal the Resource.
type jsonSource struct {
	Name   string          `json:"name"`
	Access json.RawMessage `json:"access,omitempty"`
}

// UnmarshalJSON implements a custom json unmarshal method for a Source.
func (s *Source) UnmarshalJSON(data []byte) error {
	var (
		src     Source
		jsonSrc jsonSource
	)
	if err := json.Unmarshal(data, &jsonSrc); err != nil {
		return err
	}

	src.Name = jsonSrc.Name
	acc, err := UnmarshalAccessAccessor(jsonSrc.Access)
	if err != nil {
		return err
	}
	src.Access = acc

	var sourceJSON map[string]json.RawMessage
	if err := json.Unmarshal(data, &sourceJSON); err != nil {
		return err
	}
	// remove already parsed attributes
	delete(sourceJSON, "access")
	delete(sourceJSON, "name")

	typedObjectJSONBytes, err := json.Marshal(sourceJSON)
	if err != nil {
		return err
	}
	src.TypedObjectAccessor, err = UnmarshalTypedObjectAccessor(typedObjectJSONBytes, KnownTypes{}, customCodec, nil)
	if err != nil {
		return err
	}
	*s = src
	return err
}

// MarshalJSON implements a custom json marshal method for a source.
func (s Source) MarshalJSON() ([]byte, error) {
	var (
		raw = map[string]json.RawMessage{}
		err error
	)

	raw["name"], err = json.Marshal(s.Name)
	if err != nil {
		return nil, err
	}
	raw["access"], err = MarshalAccessAccessor(s.Access)
	if err != nil {
		return nil, err
	}

	typedObjJSONBytes, err := MarshalTypedObjectAccessor(s.TypedObjectAccessor, KnownTypes{}, customCodec, nil)
	if err != nil {
		return nil, err
	}

	var typedObjectJSON map[string]json.RawMessage
	if err := json.Unmarshal(typedObjJSONBytes, &typedObjectJSON); err != nil {
		return nil, err
	}

	for key, val := range typedObjectJSON {
		raw[key] = val
	}

	return json.Marshal(raw)
}

// Resource describes a resource dependency of a component.
type Resource struct {
	ObjectMeta          `json:",inline"`
	TypedObjectAccessor `json:",inline"`

	// Relation describes the relation of the resource to the component.
	// Can be a local or external resource
	Relation ResourceRelation `json:"relation,omitempty"`

	// Access describes the type specific method to
	// access the defined resource.
	Access TypedObjectAccessor `json:"-"`
}

// jsonResource is the internal representation of a Resource
// that is used to marshal the Resource.
type jsonResource struct {
	ObjectMeta `json:",inline"`
	Relation   ResourceRelation `json:"relation,omitempty"`
	Access     json.RawMessage  `json:"access,omitempty"`
}

// UnmarshalJSON implements a custom json unmarshal method for a Resource.
func (r *Resource) UnmarshalJSON(data []byte) error {
	res := Resource{}
	jsonRes := &jsonResource{}
	if err := json.Unmarshal(data, &jsonRes); err != nil {
		return err
	}

	res.ObjectMeta = jsonRes.ObjectMeta
	res.Relation = jsonRes.Relation
	acc, err := UnmarshalAccessAccessor(jsonRes.Access)
	if err != nil {
		return err
	}
	res.Access = acc

	var resourceJSON map[string]json.RawMessage
	if err := json.Unmarshal(data, &resourceJSON); err != nil {
		return err
	}
	// remove already parsed attributes
	delete(resourceJSON, "access")
	delete(resourceJSON, "name")
	delete(resourceJSON, "version")
	delete(resourceJSON, "relation")

	typedObjectJSONBytes, err := json.Marshal(resourceJSON)
	if err != nil {
		return err
	}
	res.TypedObjectAccessor, err = UnmarshalTypedObjectAccessor(typedObjectJSONBytes, KnownTypes{}, customCodec, nil)
	if err != nil {
		return err
	}
	*r = res
	return nil
}

// MarshalJSON implements a custom json marshal method for a Resource.
func (r Resource) MarshalJSON() ([]byte, error) {
	acc, err := MarshalAccessAccessor(r.Access)
	if err != nil {
		return nil, err
	}

	jsonRes := jsonResource{
		ObjectMeta: r.ObjectMeta,
		Access:     acc,
	}

	var resourceJSON = map[string]json.RawMessage{}
	if err := remarshal(jsonRes, &resourceJSON); err != nil {
		return nil, err
	}

	typedObjJSONBytes, err := MarshalTypedObjectAccessor(r.TypedObjectAccessor, KnownTypes{}, customCodec, nil)
	if err != nil {
		return nil, err
	}

	var typedObjectJSON map[string]json.RawMessage
	if err := json.Unmarshal(typedObjJSONBytes, &typedObjectJSON); err != nil {
		return nil, err
	}

	for key, val := range typedObjectJSON {
		resourceJSON[key] = val
	}
	return json.Marshal(resourceJSON)
}

// UnmarshalTypedObjectAccessor unmarshals a type object into a valid json.
// The given known types are used to decode the data into a specific.
// The given defaultCodec is used if no matching type is known.
// An error is returned when the type is unknown and the default codec is nil.
func UnmarshalTypedObjectAccessor(data []byte, knownTypes KnownTypes, defaultCodec TypedObjectCodec, validationFunc KnownTypeValidationFunc) (TypedObjectAccessor, error) {
	accessType := &ObjectType{}
	if err := json.Unmarshal(data, accessType); err != nil {
		return nil, err
	}

	if validationFunc != nil {
		if err := validationFunc(accessType.GetType()); err != nil {
			return nil, err
		}
	}

	codec, ok := knownTypes[accessType.GetType()]
	if !ok {
		codec = defaultCodec
	}

	acc, err := codec.Decode(data)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

// MarshalTypedObjectAccessor marshals a type object into a valid json.
// The given known types are used to decode the data into a specific.
// The given defaultCodec is used if no matching type is known.
// An error is returned when the type is unknown and the default codec is nil.
func MarshalTypedObjectAccessor(acc TypedObjectAccessor, knownTypes KnownTypes, defaultCodec TypedObjectCodec, validationFunc KnownTypeValidationFunc) ([]byte, error) {
	if validationFunc != nil {
		if err := validationFunc(acc.GetType()); err != nil {
			return nil, err
		}
	}

	codec, ok := knownTypes[acc.GetType()]
	if !ok {
		codec = defaultCodec
	}

	return codec.Encode(acc)
}

// UnmarshalAccessAccessor unmarshals a json access accessor into a known go struct.
func UnmarshalAccessAccessor(data []byte) (TypedObjectAccessor, error) {
	return UnmarshalTypedObjectAccessor(data, KnownAccessTypes, customCodec, ValidateAccessType)
}

// MarshalAccessAccessor marshals a known access accessor into a valid json
func MarshalAccessAccessor(acc TypedObjectAccessor) ([]byte, error) {
	return MarshalTypedObjectAccessor(acc, KnownAccessTypes, customCodec, ValidateAccessType)
}

func remarshal(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
