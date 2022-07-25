// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsschema "github.com/gardener/landscaper/apis/schema"
)

// ExecutionManagedByLabel is the label of a deploy item that contains the name of the managed execution.
// This label is used by the extension controller to identify its managed deploy items
// todo: add conversion
const ExecutionManagedByLabel = "execution.landscaper.gardener.cloud/managed-by"

// ExecutionManagedNameLabel is the unique identifier of the deploy item managed by a execution.
// It corresponds to the execution item name.
// todo: add conversion
const ExecutionManagedNameLabel = "execution.landscaper.gardener.cloud/name"

// ExecutionDependsOnAnnotation is name of the annotation that holds the dependsOn data
// defined in the execution.
// This annotation is mainly to correctly cleanup orphaned deploy items that are not part of the execution anymore.
// todo: add conversion
const ExecutionDependsOnAnnotation = "execution.landscaper.gardener.cloud/dependsOn"

// ReconcileDeployItemsCondition is the Conditions type to indicate the deploy items status.
const ReconcileDeployItemsCondition ConditionType = "ReconcileDeployItems"

type ExecutionPhase string

const (
	ExecutionPhaseInit        = ExecutionPhase(ComponentPhaseInit)
	ExecutionPhaseProgressing = ExecutionPhase(ComponentPhaseProgressing)
	ExecutionPhaseDeleting    = ExecutionPhase(ComponentPhaseDeleting)
	ExecutionPhaseSucceeded   = ExecutionPhase(ComponentPhaseSucceeded)
	ExecutionPhaseFailed      = ExecutionPhase(ComponentPhaseFailed)
)

type ExecPhase string

const (
	ExecPhaseInit        ExecPhase = "Init"
	ExecPhaseProgressing ExecPhase = "Progressing"
	ExecPhaseCompleting  ExecPhase = "Completing"
	ExecPhaseSucceeded   ExecPhase = "Succeeded"
	ExecPhaseFailed      ExecPhase = "Failed"

	ExecPhaseInitDelete    ExecPhase = "InitDelete"
	ExecPhaseTriggerDelete ExecPhase = "TriggerDelete"
	ExecPhaseDeleting      ExecPhase = "Deleting"
	ExecPhaseDeleteFailed  ExecPhase = "DeleteFailed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExecutionList contains a list of Executions‚
type ExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Execution `json:"items"`
}

// ExecutionDefinition defines the Execution resource CRD.
var ExecutionDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "executions",
		Singular: "execution",
		ShortNames: []string{
			"exec",
		},
		Kind: "Execution",
	},
	Scope:             lsschema.NamespaceScoped,
	Storage:           true,
	Served:            true,
	SubresourceStatus: true,
	AdditionalPrinterColumns: []lsschema.CustomResourceColumnDefinition{
		{
			Name:     "Phase",
			Type:     "string",
			JSONPath: ".status.phase",
		},
		{
			Name:     "ExportRef",
			Type:     "string",
			JSONPath: ".status.exportRef.name",
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

// Execution contains the configuration of a execution and deploy item
// +kubebuilder:resource:path="executions",scope="Namespaced",shortName="exec",singular="execution"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name=Phase,type=string
// +kubebuilder:printcolumn:JSONPath=".status.exportRef.name",name=ExportRef,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
// +kubebuilder:subresource:status
type Execution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines a execution and its items
	Spec ExecutionSpec `json:"spec"`
	// Status contains the current status of the execution.
	// +optional
	Status ExecutionStatus `json:"status"`
}

// ExecutionSpec defines a execution plan.
type ExecutionSpec struct {
	// Context defines the current context of the execution.
	// +optional
	Context string `json:"context,omitempty"`

	// DeployItems defines all execution items that need to be scheduled.
	DeployItems DeployItemTemplateList `json:"deployItems,omitempty"`

	// RegistryPullSecrets defines a list of registry credentials that are used to
	// pull blueprints, component descriptors and jsonschemas from the respective registry.
	// For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// Note that the type information is used to determine the secret key and the type of the secret.
	// +optional
	RegistryPullSecrets []ObjectReference `json:"registryPullSecrets,omitempty"`

	// ReconcileID is used to update an execution even if its deploy items have not changed but their
	// reconciliation should be triggered again.
	ReconcileID string `json:"reconcileID,omitempty"`
}

// ExecutionStatus contains the current status of a execution.
type ExecutionStatus struct {
	// Phase is the current phase of the execution.
	Phase ExecutionPhase `json:"-"`

	// ObservedGeneration is the most recent generation observed for this Execution.
	// It corresponds to the Execution generation, which is updated on mutation by the landscaper.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the actual condition of a execution
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`

	// LastError describes the last error that occurred.
	// +optional
	LastError *Error `json:"lastError,omitempty"`

	// ExportReference references the object that contains the exported values.
	// only used for operation purpose.
	// +optional
	ExportReference *ObjectReference `json:"exportRef,omitempty"`

	// DeployItemReferences contain the state of all deploy items.
	// +optional
	DeployItemReferences []VersionedNamedObjectReference `json:"deployItemRefs,omitempty"`

	// ExecutionGenerations stores which generation the execution had when it last applied a specific deployitem.
	// So in this case, the observedGeneration refers to the executions generation.
	// +optional
	ExecutionGenerations []ExecutionGeneration `json:"execGenerations,omitempty"`

	// JobID is the ID of the current working request.
	JobID string `json:"jobID,omitempty"`

	// JobIDFinished is the ID of the finished working request.
	JobIDFinished string `json:"jobIDFinished,omitempty"`

	// ExecutionPhase is the current phase of the execution.
	ExecutionPhase ExecPhase `json:"phase,omitempty"`
}

// ExecutionGeneration links a deployitem to the generation of the execution when it was applied.
type ExecutionGeneration struct {
	// Name is the name of the deployitem this generation refers to.
	Name string `json:"name"`
	// ObservedGeneration stores the generation which the execution had when it last applied the referenced deployitem.
	ObservedGeneration int64 `json:"observedGeneration"`
}

// DeployItemTemplateList is a list of deploy item templates
type DeployItemTemplateList []DeployItemTemplate

// DeployItemTemplate defines a execution element that is translated into a deploy item.
// +k8s:deepcopy-gen=true
type DeployItemTemplate struct {
	// Name is the unique name of the execution.
	Name string `json:"name"`

	// DataType is the DeployItem type of the execution.
	Type DeployItemType `json:"type"`

	// Target is the object reference to the target that the deploy item should deploy to.
	// +optional
	Target *ObjectReference `json:"target,omitempty"`

	// Labels is the map of labels to be added to the deploy item.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// ProviderConfiguration contains the type specific configuration for the execution.
	Configuration *runtime.RawExtension `json:"config"`

	// DependsOn lists deploy items that need to be executed before this one
	DependsOn []string `json:"dependsOn,omitempty"`
}
