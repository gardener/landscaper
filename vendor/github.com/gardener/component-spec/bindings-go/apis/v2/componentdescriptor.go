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
	Sources []Resource `json:"sources"`
	// ComponentReferences references component dependencies that can be resolved in the current context.
	ComponentReferences []ObjectMeta `json:"componentReferences"`
	// LocalResources defines internal resources that are created by the component
	LocalResources []Resource `json:"localResources"`
	// ExternalResources defines external resources that are not produced by a third party.
	ExternalResources []Resource `json:"externalResources"`
}

// RepositoryContext describes a repository context.
type RepositoryContext struct {
	// BaseURL is the base url of the repository to resolve components.
	BaseURL string `json:"baseUrl"`
}

// ObjectMeta defines a object that is uniquely identified by its name and version.
type ObjectMeta struct {
	// Name is the context unique name of the object.
	Name string `json:"name"`
	// Version is the semver version of the object.
	Version string `json:"version"`
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

type ObjectMetaAccessor interface {
	// GetName returns the name of the access object.
	GetName() string
	// SetName sets the name of the access object.
	SetName(name string)
	// GetVersion returns the version of the access object.
	GetVersion() string
	// SetVersion sets the version of the access object.
	SetVersion(version string)
}

// AccessAccessor defines the accessor for a component
type AccessAccessor interface {
	// GetType returns the type of the access object.
	GetType() string
	// SetType sets the type of the access object.
	SetType(ttype string)
	// GetData returns the custom data of a component.
	GetData() ([]byte, error)
	// SetData sets the custom data of a component.
	SetData([]byte) error
}

// Resource describes a resource dependency of a component.
type Resource struct {
	// Version must be the same as the version of the containing component.
	ObjectMeta `json:",inline"`
	ObjectType `json:",inline"`

	// Access describes the type specific method to
	// access the defined resource.
	Access AccessAccessor `json:"-"`
}

// jsonResource is the internal representation of a Resource
// that is used to marshal the Resource.
type jsonResource struct {
	ObjectMeta `json:",inline"`
	ObjectType `json:",inline"`
	Access     json.RawMessage `json:"access,omitempty"`
}

// UnmarshalJSON implements a custom json unmarshal method for a Resource.
func (r *Resource) UnmarshalJSON(data []byte) error {
	jsonRes := &jsonResource{}
	if err := json.Unmarshal(data, &jsonRes); err != nil {
		return err
	}

	resource := &Resource{
		ObjectMeta: jsonRes.ObjectMeta,
		ObjectType: jsonRes.ObjectType,
	}

	accessType := &ObjectType{}
	if err := json.Unmarshal(jsonRes.Access, accessType); err != nil {
		return err
	}

	if err := ValidateAccessType(accessType.GetType()); err != nil {
		return err
	}

	var (
		aType AccessCodec
		ok    bool
	)
	aType, ok = KnownAccessTypes[accessType.GetType()]
	if !ok {
		aType = customCodec
	}

	acc, err := aType.Decode(jsonRes.Access)
	if err != nil {
		return err
	}

	resource.Access = acc
	*r = *resource
	return nil
}

// MarshalJSON implements a custom json marshal method for a Resource.
func (r Resource) MarshalJSON() ([]byte, error) {
	if err := ValidateAccessType(r.Access.GetType()); err != nil {
		return nil, err
	}

	var (
		encoder AccessCodec
		ok      bool
	)

	encoder, ok = KnownAccessTypes[r.Access.GetType()]
	if !ok {
		encoder = customCodec
	}

	acc, err := encoder.Encode(r.Access)
	if err != nil {
		return nil, err
	}

	jsonRes := jsonResource{
		ObjectMeta: r.ObjectMeta,
		ObjectType: r.ObjectType,
		Access:     acc,
	}

	return json.Marshal(jsonRes)
}
