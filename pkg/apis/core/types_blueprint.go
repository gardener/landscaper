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

// BlueprintResourceType is the name of the blueprint resource defined in component descriptors.
const BlueprintResourceType = "blueprint"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Blueprint contains the configuration of a component
type Blueprint struct {
	metav1.TypeMeta `json:",inline"`

	// Name is the name of the definition.
	Name string `json:"name"`

	// Version is the semver version of the definition.
	Version string `json:"version"`

	// JSONSchemaVersion defines the default jsonschema version of the blueprint.
	// e.g. "https://json-schema.org/draft/2019-09/schema"
	JSONSchemaVersion string `json:"jsonSchemaVersion"`

	// LocalTypes defines additional blueprint local schemas
	// +optional
	LocalTypes map[string]JSONSchemaDefinition `json:"localTypes,omitempty"`

	// Imports define the import values that are needed for the definition and its sub-definitions.
	// +optional
	Imports []ImportDefinition `json:"imports,omitempty"`

	// Exports define the exported values of the definition and its sub-definitions
	// +optional
	Exports []ExportDefinition `json:"exports,omitempty"`

	// BlueprintReferences define all sub-definitions that are referenced.
	// The references are relative paths to BlueprintReferences
	// +optional
	BlueprintReferences []string `json:"blueprintRefs,omitempty"`

	// DeployExecutions defines the templating executors that are sequentially executed by the landscaper.
	// The templates must return a list of deploy item templates.
	// +optional
	DeployExecutions []TemplateExecutor `json:"deployExecutions,omitempty"`

	// ExportExecutions defines the templating executors that are used to generate the exports.
	// +optional
	ExportExecutions []TemplateExecutor `json:"exportExecutions,omitempty"`
}

// ImportDefinition defines a imported value
type ImportDefinition struct {
	FieldValueDefinition `json:",inline"`

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
	ConditionalImports []ImportDefinition `json:"imports,omitempty"`
}

// ExportDefinition defines a exported value
type ExportDefinition struct {
	FieldValueDefinition `json:",inline"`
}

// FieldValueDefinition defines a im- or exported field.
type FieldValueDefinition struct {
	// Name defines the field name to search for the value and map to exports.
	// Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields
	Name string `json:"name"`
	// Schema defines the imported value as jsonschema.
	Schema JSONSchemaDefinition `json:"schema"`
}

// Default defines a default value (future idea: also reference?).
type Default struct {
	Value json.RawMessage `json:"value"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlueprintReferenceTemplate contains reference to a Blueprint included definition
// +kubebuilder:skip
type BlueprintReferenceTemplate struct {
	metav1.TypeMeta `json:",inline"`

	// Name is the unique name of the step
	Name string `json:"name"`

	// Reference defines a reference to a Blueprint.
	// The blueprint can reside in an OCI or other supported location.
	Reference ResourceReference `json:"ref"`

	// Imports defines the import mappings for the referenced component definition.
	Imports []DefinitionImportMapping `json:"imports,omitempty"`

	// Exports defines the export mappings for the referenced component definition.
	Exports []DefinitionExportMapping `json:"exports,omitempty"`

	// StaticData contains a list of data sources that are used to satisfy imports
	// +optional
	StaticData []BlueprintStaticDataSource `json:"staticData,omitempty"`
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
	OpenAPIV3Schema JSONSchemaProps `json:"openAPIV3Schema,omitempty"`
}

// BlueprintStaticDataSource defines a static data source for a blueprint
type BlueprintStaticDataSource struct {
	// Value defined inline a raw data
	// +optional
	Value json.RawMessage `json:"value,omitempty"`

	// ValueFrom defines data from an external resource
	ValueFrom *StaticDataValueFrom `json:"valueFrom,omitempty"`
}

// BlueprintStaticDataValueFrom defines static data that is read from a external resource.
type BlueprintStaticDataValueFrom struct {
	// Selects a key of a secret in the installations's namespace
	// +optional
	LocalPath string `json:"localPath,omitempty"`
}

// TemplateType describes the template mechanism.
type TemplateType string

// GOTemplateType describes the go templating type.
const GOTemplateType TemplateType = "GoTemplate"

// TemplateExecutor describes a templating mechanism and configuration.
type TemplateExecutor struct {
	// Name is the unique name of the template
	Name string `json:"name"`
	// Type describes the templating mechanism.
	Type TemplateType `json:"type"`
	// File is the path to the template in the blueprint's content.
	// +optional
	File string `json:"file,omitempty"`
	// Template contains an optional inline template.
	// The template has to be of string for go template
	// and a valid yaml/json for spiff.
	// + optional
	Template json.RawMessage `json:"template,omitempty"`
}
