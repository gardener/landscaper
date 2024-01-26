// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BlueprintResourceType is the name of the blueprint resource defined in component descriptors.
const BlueprintResourceType = "blueprint"

// ImportType is a string alias
type ImportType string

// ExportType is a string alias
type ExportType string

// ImportTypeData is the import type for data imports
const ImportTypeData = ImportType("data")

// ImportTypeTarget is the import type for target imports
const ImportTypeTarget = ImportType("target")

// ImportTypeTargetList is the import type for targetlist imports
const ImportTypeTargetList = ImportType("targetList")

// ImportTypeTargetMap is the import type for targetmap imports
const ImportTypeTargetMap = ImportType("targetMap")

// ExportTypeData is the export type for data exports
const ExportTypeData = ExportType(ImportTypeData)

// ExportTypeTarget is the export type for target exports
const ExportTypeTarget = ExportType(ImportTypeTarget)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Blueprint contains the configuration of a component
// +kubebuilder:skip
type Blueprint struct {
	metav1.TypeMeta `json:",inline"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`

	// JSONSchemaVersion defines the default jsonschema version of the blueprint.
	// e.g. "https://json-schema.org/draft/2019-09/schema"
	// +optional
	JSONSchemaVersion string `json:"jsonSchemaVersion"`

	// LocalTypes defines additional blueprint local schemas
	// +optional
	LocalTypes map[string]JSONSchemaDefinition `json:"localTypes,omitempty"`

	// Imports define the import values that are needed for the definition and its sub-definitions.
	// +optional
	Imports ImportDefinitionList `json:"imports,omitempty"`

	// ImportExecutions defines the templating executors that are sequentially executed by the landscaper.
	// The templates must return a list of errors
	// +optional
	ImportExecutions []TemplateExecutor `json:"importExecutions,omitempty"`

	// Exports define the exported values of the definition and its sub-definitions
	// +optional
	Exports ExportDefinitionList `json:"exports,omitempty"`

	// Subinstallations defines an optional list of subinstallations (for aggregating blueprints).
	// +optional
	Subinstallations SubinstallationTemplateList `json:"subinstallations,omitempty"`

	// SubinstallationExecutions defines the templating executors that are sequentially executed by the landscaper.
	// The templates must return a list of installation templates.
	// Both subinstallations and SubinstallationExecutions are valid options and will be merged.
	// +optional
	SubinstallationExecutions []TemplateExecutor `json:"subinstallationExecutions,omitempty"`

	// DeployExecutions defines the templating executors that are sequentially executed by the landscaper.
	// The templates must return a list of deploy item templates.
	// +optional
	DeployExecutions []TemplateExecutor `json:"deployExecutions,omitempty"`

	// ExportExecutions defines the templating executors that are used to generate the exports.
	// +optional
	ExportExecutions []TemplateExecutor `json:"exportExecutions,omitempty"`
}

// ImportDefinitionList defines a list of import defiinitions.
type ImportDefinitionList []ImportDefinition

// ImportDefinition defines a imported value
type ImportDefinition struct {
	FieldValueDefinition `json:",inline"`

	// Type specifies which kind of object is being imported.
	// This field should be set and will likely be mandatory in future.
	// +optional
	Type ImportType `json:"type,omitempty"`

	// Required specifies whether the import is required for the component to run.
	// Defaults to true.
	// +optional
	Required *bool `json:"required"`

	// Default sets a default value for the current import that is used if the key is not set.
	Default Default `json:"default,omitempty"`

	// ConditionalImports are Imports that are only valid if this imports is satisfied.
	// Does only make sense for optional imports.
	// +optional
	ConditionalImports ImportDefinitionList `json:"imports,omitempty"`
}

// ExportDefinitionList defines a list of export definitions.
type ExportDefinitionList []ExportDefinition

// ExportDefinition defines a exported value
type ExportDefinition struct {
	FieldValueDefinition `json:",inline"`

	// Type specifies which kind of object is being exported.
	// This field should be set and will likely be mandatory in future.
	// +optional
	Type ExportType `json:"type,omitempty"`
}

// FieldValueDefinition defines a im- or exported field.
// Either schema or target type have to be defined
type FieldValueDefinition struct {
	// Name defines the field name to search for the value and map to exports.
	// Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields
	Name string `json:"name"`
	// Schema defines the imported value as jsonschema.
	// +optional
	Schema *JSONSchemaDefinition `json:"schema,omitempty"`
	// TargetType defines the type of the imported target.
	// +optional
	TargetType string `json:"targetType,omitempty"`
}

// Default defines a default value (future idea: also reference?).
type Default struct {
	Value AnyJSON `json:"value"`
}

// BlueprintStaticDataSource defines a static data source for a blueprint
type BlueprintStaticDataSource struct {
	// Value defined inline a raw data
	// +optional
	Value AnyJSON `json:"value,omitempty"`

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

// SpiffTemplateType describes the spiff templating type.
const SpiffTemplateType TemplateType = "Spiff"

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
	// and either a string or valid yaml/json for spiff.
	// + optional
	Template AnyJSON `json:"template,omitempty"`
}

// SubinstallationTemplateList is a list of installation templates
type SubinstallationTemplateList []SubinstallationTemplate

// SubinstallationTemplate defines a subinstallation template.
type SubinstallationTemplate struct {
	// File references a subinstallation template stored in another file.
	// +optional
	File string `json:"file,omitempty"`

	// An inline subinstallation template.
	// +optional
	*InstallationTemplate `json:",inline"`
}
