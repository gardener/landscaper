// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"
)

// EncompassedByLabel is the label that contains the name of the parent installation
// that encompasses the current installation.
// todo: add conversion
const EncompassedByLabel = "landscaper.gardener.cloud/encompassed-by"

// SubinstallationNameAnnotation is the annotation that contains the name of the subinstallation.
// todo: add conversion
const SubinstallationNameAnnotation = "landscaper.gardener.cloud/subinstallation-name"

// todo: keep only subinstallations?
const KeepChildrenAnnotation = "landscaper.gardener.cloud/keep-children"

// EnsureSubInstallationsCondition is the Conditions type to indicate the sub installation status.
const EnsureSubInstallationsCondition ConditionType = "EnsureSubInstallations"

// ReconcileExecutionCondition is the Conditions type to indicate the execution reconcile status.
const ReconcileExecutionCondition ConditionType = "ReconcileExecution"

// ValidateImportsCondition is the Conditions type to indicate status of the import validation.
const ValidateImportsCondition ConditionType = "ValidateImports"

// CreateImportsCondition is the Conditions type to indicate status of the imported data and data objects.
const CreateImportsCondition ConditionType = "CreateImports"

// CreateExportsCondition is the Conditions type to indicate status of the exported data and data objects.
const CreateExportsCondition ConditionType = "CreateExports"

// EnsureExecutionsCondition is the Conditions type to indicate the executions status.
const EnsureExecutionsCondition ConditionType = "EnsureExecutions"

// ValidateExportCondition is the Conditions type to indicate validation status of teh exported data.
const ValidateExportCondition ConditionType = "ValidateExport"

// ComponentReferenceOverwriteCondition is the Conditions type to indicate that the component reference was overwritten.
const ComponentReferenceOverwriteCondition ConditionType = "ComponentReferenceOverwrite"

type ComponentInstallationPhase string

const (
	ComponentPhaseInit        ComponentInstallationPhase = "Init"
	ComponentPhasePending     ComponentInstallationPhase = "PendingDependencies"
	ComponentPhaseProgressing ComponentInstallationPhase = "Progressing"
	ComponentPhaseDeleting    ComponentInstallationPhase = "Deleting"
	ComponentPhaseAborted     ComponentInstallationPhase = "Aborted"
	ComponentPhaseSucceeded   ComponentInstallationPhase = "Succeeded"
	ComponentPhaseFailed      ComponentInstallationPhase = "Failed"
)

type InstallationPhase string

const (
	InstallationPhaseInit           InstallationPhase = "Init"
	InstallationPhaseObjectsCreated InstallationPhase = "ObjectsCreated"
	InstallationPhaseProgressing    InstallationPhase = "Progressing"
	InstallationPhaseCompleting     InstallationPhase = "Completing"
	InstallationPhaseSucceeded      InstallationPhase = "Succeeded"
	InstallationPhaseFailed         InstallationPhase = "Failed"

	InstallationPhaseInitDelete    InstallationPhase = "InitDelete"
	InstallationPhaseTriggerDelete InstallationPhase = "TriggerDelete"
	InstallationPhaseDeleting      InstallationPhase = "Deleting"
	InstallationPhaseDeleteFailed  InstallationPhase = "DeleteFailed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstallationList contains a list of Components
type InstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Installation `json:"items"`
}

// InstallationDefinition defines the Installation resource CRD.
var InstallationDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "installations",
		Singular: "installation",
		ShortNames: []string{
			"inst",
		},
		Kind: "Installation",
	},
	Scope:             lsschema.NamespaceScoped,
	Storage:           true,
	Served:            true,
	SubresourceStatus: true,
	AdditionalPrinterColumns: []lsschema.CustomResourceColumnDefinition{
		{
			Name:     "phase",
			Type:     "string",
			JSONPath: ".status.phase",
		},
		{
			Name:     "Execution",
			Type:     "string",
			JSONPath: ".status.executionRef.name",
		},
		{
			Name:     "Age",
			Type:     "date",
			JSONPath: ".metadata.creationTimestamp",
		},
	},
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Installation contains the configuration of a component
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
}

// InstallationStatus contains the current status of a Installation.
type InstallationStatus struct {
	// Phase is the current phase of the installation.
	Phase ComponentInstallationPhase `json:"-"`

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
}

// InstallationImports defines import of data objects and targets.
type InstallationImports struct {
	// Data defines all data object imports.
	// +optional
	Data []DataImport `json:"data,omitempty"`

	// Targets defines all target imports.
	// +optional
	Targets []TargetImport `json:"targets,omitempty"`

	// ComponentDescriptors defines all component descriptor imports.
	// +optional
	ComponentDescriptors []ComponentDescriptorImport `json:"componentDescriptors,omitempty"`
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
	// The reference can also be a namespaces name. E.g. "default/mydataref"
	// +optional
	DataRef string `json:"dataRef,omitempty"`

	// Version specifies the imported data version.
	// defaults to "v1"
	// +optional
	Version string `json:"version,omitempty"`

	// SecretRef defines a data reference from a secret.
	// This method is not allowed in installation templates.
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// ConfigMapRef defines a data reference from a configmap.
	// This method is not allowed in installation templates.
	// +optional
	ConfigMapRef *ConfigMapReference `json:"configMapRef,omitempty"`
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

type ComponentDescriptorImport struct {
	// Name the internal name of the imported/exported component descriptor.
	Name string `json:"name"`

	// Ref is a reference to a component descriptor in a registry.
	// Exactly one of Ref, SecretRef, ConfigMapRef, and List has to be specified.
	// +optional
	Ref *ComponentDescriptorReference `json:"ref,omitempty"`

	// SecretRef is a reference to a key in a secret in the cluster.
	// Exactly one of Ref, SecretRef, ConfigMapRef, and List has to be specified.
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// ConfigMapRef is a reference to a key in a config map in the cluster.
	// Exactly one of Ref, SecretRef, ConfigMapRef, and List has to be specified.
	// +optional
	ConfigMapRef *ConfigMapReference `json:"configMapRef,omitempty"`

	// List represents a list of component descriptor imports.
	// Exactly one of Ref, SecretRef, ConfigMapRef, and List has to be specified.
	// +optional
	List []ComponentDescriptorImportData `json:"list,omitempty"`

	// DataRef can be used to reference component descriptors imported by the parent installation.
	// This field is used in subinstallation templates only, use one of the other fields instead for root installations.
	// +optional
	DataRef string `json:"dataRef,omitempty"`
}

type ComponentDescriptorImportData struct {
	// Ref is a reference to a component descriptor in a registry.
	// Exactly one of Ref, SecretRef, ConfigMapRef, and List has to be specified.
	// +optional
	Ref *ComponentDescriptorReference `json:"ref,omitempty"`

	// SecretRef is a reference to a key in a secret in the cluster.
	// Exactly one of Ref, SecretRef, ConfigMapRef, and List has to be specified.
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// ConfigMapRef is a reference to a key in a config map in the cluster.
	// Exactly one of Ref, SecretRef, ConfigMapRef, and List has to be specified.
	// +optional
	ConfigMapRef *ConfigMapReference `json:"configMapRef,omitempty"`

	// DataRef can be used to reference component descriptors imported by the parent installation.
	// This field is used in subinstallation templates only, use one of the other fields instead for root installations.
	// +optional
	DataRef string `json:"dataRef,omitempty"`
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

// InlineBlueprint defines a inline blueprint with component descriptor and
// filesystem.
type InlineBlueprint struct {
	// Filesystem defines a inline yaml filesystem with a blueprint.
	Filesystem AnyJSON `json:"filesystem"`
}

// ComponentDescriptorDefinition defines the component descriptor that should be used
// for the installation
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
	// CDImportStatusType is an ImportStatusType for component descriptors
	CDImportStatusType ImportStatusType = "componentDescriptor"
	// CDListImportStatusType is an ImportStatusType for component descriptor lists
	CDListImportStatusType ImportStatusType = "componentDescriptorList"
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

// CDImportStatus is the import status of a component descriptor
type CDImportStatus struct {
	// ComponentDescriptorRef is a reference to a component descriptor
	// +optional
	ComponentDescriptorRef *ComponentDescriptorReference `json:"componentDescriptorRef,omitempty"`
	// SecretRef is the name of the secret.
	// +optional
	SecretRef string `json:"secretRef,omitempty"`
	// ConfigMapRef is the name of the imported configmap.
	// +optional
	ConfigMapRef string `json:"configMapRef,omitempty"`
	// SourceRef is the reference to the installation from where the value is imported
	SourceRef *ObjectReference `json:"sourceRef,omitempty"`
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
	// ComponentDescriptorRef is a reference to a component descriptor
	// +optional
	ComponentDescriptorRef *ComponentDescriptorReference `json:"componentDescriptorRef,omitempty"`
	// ComponentDescriptors is a list of import statuses for component descriptors
	// +optional
	ComponentDescriptors []CDImportStatus `json:"componentDescriptorList,omitempty"`
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

// MarshalJSON implements the json marshaling for a TargetImport
// Why this is needed:
//   We need Targets to not have the 'omitempty' annotation,
//   because the code distinguishes between nil and an empty list.
//   Not having the annotation causes the default json marshal to write
//   'null' in case of nil, which causes problems.
func (ti TargetImport) MarshalJSON() ([]byte, error) {

	type TargetImportWithTargets struct {
		Name                string   `json:"name"`
		Target              string   `json:"target,omitempty"`
		Targets             []string `json:"targets"`
		TargetListReference string   `json:"targetListRef,omitempty"`
	}
	type TargetImportWithoutTargets struct {
		Name                string   `json:"name"`
		Target              string   `json:"target,omitempty"`
		Targets             []string `json:"targets,omitempty"`
		TargetListReference string   `json:"targetListRef,omitempty"`
	}

	if ti.Targets == nil {
		return json.Marshal(TargetImportWithoutTargets(ti))
	}
	return json.Marshal(TargetImportWithTargets(ti))
}

// MarshalJSON implements the json marshaling for a ComponentDescriptorImport
// This is needed for the same reasons as the MarshalJSON function for TargetImports.
func (cdi ComponentDescriptorImport) MarshalJSON() ([]byte, error) {

	type ComponentDescriptorImportWithCDs struct {
		Name         string                          `json:"name"`
		Ref          *ComponentDescriptorReference   `json:"ref,omitempty"`
		SecretRef    *SecretReference                `json:"secretRef,omitempty"`
		ConfigMapRef *ConfigMapReference             `json:"configMapRef,omitempty"`
		List         []ComponentDescriptorImportData `json:"list"`
		DataRef      string                          `json:"dataRef,omitempty"`
	}

	type ComponentDescriptorImportWithoutCDs struct {
		Name         string                          `json:"name"`
		Ref          *ComponentDescriptorReference   `json:"ref,omitempty"`
		SecretRef    *SecretReference                `json:"secretRef,omitempty"`
		ConfigMapRef *ConfigMapReference             `json:"configMapRef,omitempty"`
		List         []ComponentDescriptorImportData `json:"list,omitempty"`
		DataRef      string                          `json:"dataRef,omitempty"`
	}

	if cdi.List == nil {
		return json.Marshal(ComponentDescriptorImportWithoutCDs(cdi))
	}
	return json.Marshal(ComponentDescriptorImportWithCDs(cdi))
}
