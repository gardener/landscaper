// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package core

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExecutionType defines the type of the execution
type ExecutionType string

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentDefinition contains the configuration of a component
type ComponentDefinition struct {
	metav1.TypeMeta `json:",inline"`

	// Name is the name of the definition.
	Name string `json:"name"`

	// Version is the semver version of the definition.
	Version string `json:"version"`

	// CustomTypes defines additional DataTypes
	// +optional
	CustomTypes []CustomType `json:"customTypes,omitempty"`

	// Imports define the import values that are needed for the definition and its sub-definitions.
	// +optional
	Imports []DefinitionImport `json:"imports,omitempty"`

	// Exports define the exported values of the definition and its sub-definitions
	// +optional
	Exports []DefinitionExport `json:"exports,omitempty"`

	// DefinitionReferences define all sub-definitions that are referenced.
	// +optional
	DefinitionReferences []DefinitionReference `json:"definitionRefs,omitempty"`

	// DeployItemReferences defines the executors that are sequentially executed by the landscaper
	// +optional
	Executors string `json:"executors"`
}

// DefinitionImport defines a imported value
type DefinitionImport struct {
	DefinitionFieldValue `json:",inline"`

	// Required specifies whether the import is required for the component to run.
	// Defaults to true.
	// +optional
	Required *bool `json:"required"`

	// Default sets a default value for the current import that is used if the key is not set.
	Default Default `json:"default,omitempty"`

	// ConditionalImports are Imports that are only valid if this imports is satisfied.
	// Does only make sense for optional imports.
	// todo: maybe restrict only for required=false
	// todo: see if this works with recursion
	// +optional
	ConditionalImports []DefinitionImport `json:"imports,omitempty"`
}

// DefinitionExport defines a exported value
type DefinitionExport struct {
	DefinitionFieldValue `json:",inline"`
}

// DefinitionExport defines a exported value
type DefinitionFieldValue struct {
	// Key defines the field name to search for the value and map to exports.
	// Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields
	Key string `json:"key"`
	// DataType is the data type of the imported value
	Type string `json:"type"`
}

// Default defines a default value (future idea: also reference?).
type Default struct {
	Value json.RawMessage `json:"value"`
}

// DefinitionReference defines a referenced child component definition.
type DefinitionReference struct {
	// Name is the unique name of the step
	Name string `json:"name"`

	// Reference defines a reference to a ComponentDefinition.
	// The definition can reside in an OCI or other supported location.
	Reference string `json:"ref"`

	// Imports defines the import mappings for the referenced component definition.
	Imports []DefinitionImportMapping `json:"imports,omitempty"`

	// Exports defines the export mappings for the referenced component definition.
	Exports []DefinitionExportMapping `json:"exports,omitempty"`
}

// DefinitionImportMapping defines the mapping of import value
// to the import of the referenced definition.
type DefinitionImportMapping struct {
	DefinitionFieldMapping `json:",inline"`
}

// DefinitionExportMapping defines the mapping of export value of the referenced definition
// to the export value of this definition.
type DefinitionExportMapping struct {
	DefinitionFieldMapping `json:",inline"`
}

// DefinitionFieldMapping defines a mapping of a field name to another.
type DefinitionFieldMapping struct {
	// From defines a field name to get a value from an import.
	From string `json:"from"`

	// To defines a field name to map the value from the "from" field.
	To string `json:"to"`
}

// CustomType defines a custom datatype.
type CustomType struct {
	// Name is the unique name of the datatype
	Name string `json:"name"`

	// OpenAPIV3Schema defines the type as openapi v3 scheme.
	OpenAPIV3Schema OpenAPIV3Schema `json:"openAPIV3Schema,omitempty"`
}
