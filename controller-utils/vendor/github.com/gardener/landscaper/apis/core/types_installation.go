// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallationPhase is a string that contains the installation phase
type InstallationPhase string

// InstallationDeletionPhase is a string that contains the deletion phase
type InstallationDeletionPhase string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstallationList contains a list of Components
type InstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Installation `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Installation contains the configuration of a installation
type Installation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the specification for a installation.
	Spec InstallationSpec `json:"spec"`

	// Status contains the status of the installation.
	// +optional
	Status InstallationStatus `json:"status"`
}

// InstallationSpec defines a component installation.
type InstallationSpec struct {
	// Context defines the current context of the installation.
	// +optional
	Context string `json:"context,omitempty"`

	//ComponentDescriptor is a reference to the installation's component descriptor
	// +optional
	ComponentDescriptor *ComponentDescriptorDefinition `json:"componentDescriptor,omitempty"`

	// Blueprint is the resolved reference to the definition.
	Blueprint BlueprintDefinition `json:"blueprint"`

	// RegistryPullSecrets defines a list of registry credentials that are used to
	// pull blueprints, component descriptors and jsonschemas from the respective registry.
	// For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// Note that the type information is used to determine the secret key and the type of the secret.
	// +optional
	RegistryPullSecrets []ObjectReference `json:"registryPullSecrets,omitempty"`

	// Imports define the imported data objects and targets.
	// +optional
	Imports InstallationImports `json:"imports,omitempty"`

	// ImportDataMappings contains a template for restructuring imports.
	// It is expected to contain a key for every blueprint-defined data import.
	// Missing keys will be defaulted to their respective data import.
	// Example: namespace: (( installation.imports.namespace ))
	// +optional
	ImportDataMappings map[string]AnyJSON `json:"importDataMappings,omitempty"`

	// Exports define the exported data objects and targets.
	// +optional
	Exports InstallationExports `json:"exports,omitempty"`

	// ExportDataMappings contains a template for restructuring exports.
	// It is expected to contain a key for every blueprint-defined data export.
	// Missing keys will be defaulted to their respective data export.
	// Example: namespace: (( blueprint.exports.namespace ))
	// +optional
	ExportDataMappings map[string]AnyJSON `json:"exportDataMappings,omitempty"`

	// AutomaticReconcile allows to configure automatically repeated reconciliations.
	// +optional
	AutomaticReconcile *AutomaticReconcile `json:"automaticReconcile,omitempty"`
}

// AutomaticReconcile allows to configure automatically repeated reconciliations.
type AutomaticReconcile struct {
	// SucceededReconcile allows to configure automatically repeated reconciliations for succeeded installations.
	// If not set, no such automatically repeated reconciliations are triggered.
	// +optional
	SucceededReconcile *SucceededReconcile `json:"succeededReconcile,omitempty"`

	// FailedReconcile allows to configure automatically repeated reconciliations for failed installations.
	// If not set, no such automatically repeated reconciliations are triggered.
	// +optional
	FailedReconcile *FailedReconcile `json:"failedReconcile,omitempty"`
}

// SucceededReconcile allows to configure automatically repeated reconciliations for succeeded installations
type SucceededReconcile struct {
	// Interval specifies the interval between two subsequent repeated reconciliations. If not set, a default of
	// 24 hours is used.
	// +optional
	Interval *Duration `json:"interval,omitempty"`
}

// FailedReconcile allows to configure automatically repeated reconciliations for failed installations
type FailedReconcile struct {
	// NumberOfReconciles specifies the maximal number of automatically repeated reconciliations. If not set, no upper
	// limit exists.
	// +optional
	NumberOfReconciles *int `json:"numberOfReconciles,omitempty"`

	// Interval specifies the interval between two subsequent repeated reconciliations. If not set, a default
	// of 5 minutes is used.
	// +optional
	Interval *Duration `json:"interval,omitempty"`
}

// InstallationStatus contains the current status of a Installation.
type InstallationStatus struct {
	// ObservedGeneration is the most recent generation observed for this ControllerInstallations.
	// It corresponds to the ControllerInstallations generation, which is updated on mutation by the landscaper.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the actual condition of a installation
	Conditions []Condition `json:"conditions,omitempty"`

	// LastError describes the last error that occurred.
	LastError *Error `json:"lastError,omitempty"`

	// ConfigGeneration is the generation of the exported values.
	ConfigGeneration string `json:"configGeneration"`

	// Imports contain the state of the imported values.
	Imports []ImportStatus `json:"imports,omitempty"`

	// InstallationReferences contain all references to sub-components
	// that are created based on the component definition.
	InstallationReferences []NamedObjectReference `json:"installationRefs,omitempty"`

	// ExecutionReference is the reference to the execution that schedules the templated execution items.
	ExecutionReference *ObjectReference `json:"executionRef,omitempty"`

	// JobID is the ID of the current working request.
	JobID string `json:"jobID,omitempty"`

	// JobIDFinished is the ID of the finished working request.
	JobIDFinished string `json:"jobIDFinished,omitempty"`

	// InstallationPhase is the current phase of the installation.
	InstallationPhase InstallationPhase `json:"phase,omitempty"`

	// ImportsHash is the hash of the import data.
	ImportsHash string `json:"importsHash,omitempty"`

	// AutomaticReconcileStatus describes the status of automatically triggered reconciles.
	// +optional
	AutomaticReconcileStatus *AutomaticReconcileStatus `json:"automaticReconcileStatus,omitempty"`
}

// AutomaticReconcileStatus describes the status of automatically triggered reconciles.
type AutomaticReconcileStatus struct {
	// Generation describes the generation of the installation for which the status holds.
	// +optional
	Generation int64 `json:"generation,omitempty"`
	// NumberOfReconciles is the number of automatic reconciles for the installation with the stored generation.
	// +optional
	NumberOfReconciles int `json:"numberOfReconciles,omitempty"`
	// LastReconcileTime is the time of the last automatically triggered reconcile.
	// +optional
	LastReconcileTime metav1.Time `json:"lastReconcileTime,omitempty"`
	// OnFailed is true if the last automatically triggered reconcile was done for a failed installation.
	// +optional
	OnFailed bool `json:"onFailed,omitempty"`
}

// InstallationImports defines import of data objects and targets.
type InstallationImports struct {
	// Data defines all data object imports.
	// +optional
	Data []DataImport `json:"data,omitempty"`

	// Targets defines all target imports.
	// +optional
	Targets []TargetImport `json:"targets,omitempty"`
}

// InstallationExports defines exports of data objects and targets.
type InstallationExports struct {
	// Data defines all data object exports.
	// +optional
	Data []DataExport `json:"data,omitempty"`

	// Targets defines all target exports.
	// +optional
	Targets []TargetExport `json:"targets,omitempty"`
}

// DataImport is a data object import.
type DataImport struct {
	// Name the internal name of the imported/exported data.
	Name string `json:"name"`

	// DataRef is the name of the in-cluster data object.
	DataRef string `json:"dataRef"`

	// Version specifies the imported data version.
	// defaults to "v1"
	// +optional
	Version string `json:"version,omitempty"`

	// SecretRef defines a data reference from a secret.
	// This method is not allowed in installation templates.
	// +optional
	SecretRef *LocalSecretReference `json:"secretRef,omitempty"`

	// ConfigMapRef defines a data reference from a configmap.
	// This method is not allowed in installation templates.
	// +optional
	ConfigMapRef *LocalConfigMapReference `json:"configMapRef,omitempty"`
}

// DataExport is a data object export.
type DataExport struct {
	// Name the internal name of the imported/exported data.
	Name string `json:"name"`

	// DataRef is the name of the in-cluster data object.
	DataRef string `json:"dataRef"`
}

// TargetImport is either a single target or a target list import.
type TargetImport struct {
	// Name the internal name of the imported target.
	Name string `json:"name"`

	// Target is the name of the in-cluster target object.
	// Exactly one of Target, Targets, and TargetListReference has to be specified.
	// +optional
	Target string `json:"target,omitempty"`

	// Targets is a list of in-cluster target objects.
	// Exactly one of Target, Targets, and TargetListReference has to be specified.
	// +optional
	Targets []string `json:"targets"`

	// TargetListReference can (only) be used to import a targetlist that has been imported by the parent installation.
	// Exactly one of Target, Targets, and TargetListReference has to be specified.
	// +optional
	TargetListReference string `json:"targetListRef,omitempty"`
}

// TargetExport is a single target export.
type TargetExport struct {
	// Name the internal name of the exported target.
	Name string `json:"name"`

	// Target is the name of the in-cluster target object.
	// +optional
	Target string `json:"target,omitempty"`
}

// BlueprintDefinition defines the blueprint that should be used for the installation.
type BlueprintDefinition struct {
	// Reference defines a remote reference to a blueprint
	// +optional
	Reference *RemoteBlueprintReference `json:"ref,omitempty"`
	// Inline defines a inline yaml filesystem with a blueprint.
	// +optional
	Inline *InlineBlueprint `json:"inline,omitempty"`
}

// RemoteBlueprintReference describes a reference to a blueprint defined by a component descriptor.
type RemoteBlueprintReference struct {
	// ResourceName is the name of the blueprint as defined by a component descriptor.
	ResourceName string `json:"resourceName"`
}

// InlineBlueprint defines an inline blueprint with component descriptor and
// filesystem.
type InlineBlueprint struct {
	// Filesystem defines a inline yaml filesystem with a blueprint.
	Filesystem AnyJSON `json:"filesystem"`
}

// ComponentDescriptorDefinition defines the component descriptor that should be used
// for the installatoin
type ComponentDescriptorDefinition struct {
	// ComponentDescriptorReference is the reference to a component descriptor
	// +optional
	Reference *ComponentDescriptorReference `json:"ref,omitempty"`

	// InlineDescriptorReference defines an inline component descriptor
	// +optional
	Inline *cdv2.ComponentDescriptor `json:"inline,omitempty"`
}

// ComponentDescriptorReference is the reference to a component descriptor.
// given an optional context.
type ComponentDescriptorReference struct {
	// RepositoryContext defines the context of the component repository to resolve blueprints.
	// +optional
	RepositoryContext *cdv2.UnstructuredTypedObject `json:"repositoryContext,omitempty"`
	// ComponentName defines the unique of the component containing the resource.
	ComponentName string `json:"componentName"`
	// Version defines the version of the component.
	Version string `json:"version"`
}

// ObjectMeta returns the component descriptor v2 compatible object meta for a resource reference.
func (r ComponentDescriptorReference) ObjectMeta() cdv2.ObjectMeta {
	return cdv2.ObjectMeta{
		Name:    r.ComponentName,
		Version: r.Version,
	}
}

// StaticDataSource defines a static data source
type StaticDataSource struct {
	// Value defined inline a raw data
	// +optional
	Value AnyJSON `json:"value,omitempty"`

	// ValueFrom defines data from an external resource
	ValueFrom *StaticDataValueFrom `json:"valueFrom,omitempty"`
}

// StaticDataValueFrom defines a static data that is read from a external resource.
type StaticDataValueFrom struct {
	// Selects a key of a secret in the installations's namespace
	// +optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`

	// Selects a key from multiple secrets in the installations's namespace
	// that matches the given labels.
	// +optional
	SecretLabelSelector *SecretLabelSelectorRef `json:"secretLabelSelector,omitempty"`
}

// SecretLabelSelectorRef selects secrets with the given label and key.
type SecretLabelSelectorRef struct {
	// Selector is a map of labels to select specific secrets.
	Selector map[string]string `json:"selector"`

	// The key of the secret to select from.  Must be a valid secret key.
	Key string `json:"key"`
}

// ImportStatusType defines the type of a import status.
type ImportStatusType string

const (
	// DataImportStatusType is an ImportStatusType for data objects
	DataImportStatusType ImportStatusType = "dataobject"
	// TargetImportStatusType is an ImportStatusType for targets
	TargetImportStatusType ImportStatusType = "target"
	// TargetListImportStatusType is an ImportStatusType for target lists
	TargetListImportStatusType ImportStatusType = "targetList"
)

// TargetImportStatus
type TargetImportStatus struct {
	// Target is the name of the in-cluster target object.
	Target string `json:"target,omitempty"`
	// SourceRef is the reference to the installation from where the value is imported
	SourceRef *ObjectReference `json:"sourceRef,omitempty"`
	// ConfigGeneration is the generation of the imported value.
	ConfigGeneration string `json:"configGeneration,omitempty"`
}

// ImportStatus hold the state of a import.
type ImportStatus struct {
	// Name is the distinct identifier of the import.
	// Can be either from data or target imports
	Name string `json:"name"`
	// Type defines the kind of import.
	// Can be either DataObject, Target, or TargetList
	Type ImportStatusType `json:"type"`
	// Target is the name of the in-cluster target object.
	// +optional
	Target string `json:"target,omitempty"`
	// TargetList is a list of import statuses for in-cluster target objects.
	// +optional
	Targets []TargetImportStatus `json:"targetList,omitempty"`
	// DataRef is the name of the in-cluster data object.
	// +optional
	DataRef string `json:"dataRef,omitempty"`
	// SecretRef is the name of the secret.
	// +optional
	SecretRef string `json:"secretRef,omitempty"`
	// ConfigMapRef is the name of the imported configmap.
	// +optional
	ConfigMapRef string `json:"configMapRef,omitempty"`
	// SourceRef is the reference to the installation from where the value is imported
	// +optional
	SourceRef *ObjectReference `json:"sourceRef,omitempty"`
	// ConfigGeneration is the generation of the imported value.
	// +optional
	ConfigGeneration string `json:"configGeneration,omitempty"`
}
