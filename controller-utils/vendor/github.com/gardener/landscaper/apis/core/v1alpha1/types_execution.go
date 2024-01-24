// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsschema "github.com/gardener/landscaper/apis/schema"
	"github.com/gardener/landscaper/apis/utils"
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

func (p ExecutionPhase) String() string {
	return string(p)
}

func (p ExecutionPhase) IsFinal() bool {
	switch p {
	case ExecutionPhases.Succeeded, ExecutionPhases.Failed, ExecutionPhases.DeleteFailed:
		return true
	}
	return false
}

func (p ExecutionPhase) IsDeletion() bool {
	switch p {
	case ExecutionPhases.InitDelete, ExecutionPhases.TriggerDelete, ExecutionPhases.Deleting, ExecutionPhases.DeleteFailed:
		return true
	}
	return false
}

func (p ExecutionPhase) IsFailed() bool {
	switch p {
	case ExecutionPhases.Failed, ExecutionPhases.DeleteFailed:
		return true
	}
	return false
}

func (p ExecutionPhase) IsEmpty() bool {
	return p.String() == ""
}

var (
	ExecutionPhases = struct {
		Init,
		Progressing,
		Completing,
		Succeeded,
		Failed,
		InitDelete,
		TriggerDelete,
		Deleting,
		DeleteFailed ExecutionPhase
	}{
		Init:          ExecutionPhase(PhaseStringInit),
		Progressing:   ExecutionPhase(PhaseStringProgressing),
		Completing:    ExecutionPhase(PhaseStringCompleting),
		Succeeded:     ExecutionPhase(PhaseStringSucceeded),
		Failed:        ExecutionPhase(PhaseStringFailed),
		InitDelete:    ExecutionPhase(PhaseStringInitDelete),
		TriggerDelete: ExecutionPhase(PhaseStringTriggerDelete),
		Deleting:      ExecutionPhase(PhaseStringDeleting),
		DeleteFailed:  ExecutionPhase(PhaseStringDeleteFailed),
	}
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

	// DeployItemsCompressed as zipped byte array
	DeployItemsCompressed []byte `json:"deployItemsCompressed,omitempty"`
}

// ExecutionStatus contains the current status of a execution.
type ExecutionStatus struct {
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

	// DeployItemCache contains the currently existing deploy item belonging to the execution. If nil undefined.
	// +optional
	DeployItemCache *DeployItemCache `json:"deployItemCache,omitempty"`

	// JobID is the ID of the current working request.
	JobID string `json:"jobID,omitempty"`

	// JobIDFinished is the ID of the finished working request.
	JobIDFinished string `json:"jobIDFinished,omitempty"`

	// ExecutionPhase is the current phase of the execution.
	ExecutionPhase ExecutionPhase `json:"phase,omitempty"`

	// PhaseTransitionTime is the time when the phase last changed.
	// +optional
	PhaseTransitionTime *metav1.Time `json:"phaseTransitionTime,omitempty"`

	// TransitionTimes contains timestamps of status transitions
	// +optional
	TransitionTimes *TransitionTimes `json:"transitionTimes,omitempty"`
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

	// Timeout specifies how long the deployer may take to apply the deploy item.
	// When the time is exceeded, the deploy item fails.
	// Value has to be parsable by time.ParseDuration (or 'none' to deactivate the timeout).
	// Defaults to ten minutes if not specified.
	// +optional
	Timeout *Duration `json:"timeout,omitempty"`

	// UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed.
	// +optional
	UpdateOnChangeOnly bool `json:"updateOnChangeOnly,omitempty"`

	// OnDelete specifies particular setting when deleting a deploy item
	OnDelete *OnDeleteConfig `json:"onDelete,omitempty"`
}

// OnDeleteConfig specifies particular setting when deleting a deploy item
type OnDeleteConfig struct {
	// SkipUninstallIfClusterRemoved specifies that uninstall is skipped if the target cluster is already deleted.
	// Works only in the context of an existing target sync object which is used to check the Garden project with
	// the shoot cluster resources
	SkipUninstallIfClusterRemoved bool `json:"skipUninstallIfClusterRemoved,omitempty"`
}

func (r *ExecutionSpec) UnmarshalJSON(data []byte) error {
	type Alias ExecutionSpec
	a := (*Alias)(r)

	if err := json.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("unable to unmarshal execution spec: %w", err)
	}

	if len(a.DeployItemsCompressed) > 0 {
		diBytes, err := utils.Gunzip(a.DeployItemsCompressed)
		if err != nil {
			return fmt.Errorf("unable to gunzip deployitems of execution spec: %w", err)
		}

		deployItems := DeployItemTemplateList{}
		if err := json.Unmarshal(diBytes, &deployItems); err != nil {
			return fmt.Errorf("unable to unmarshal deployitems of execution spec: %w", err)
		}

		a.DeployItemsCompressed = nil
		a.DeployItems = deployItems
	}

	return nil
}

func (r ExecutionSpec) MarshalJSON() ([]byte, error) {
	// Copy the ExecutionSpec type. The copied type has the standard marshaling behaviour,
	// whereas ExecutionSpec has the custom marshaling behaviour defined in this method.
	type Alias ExecutionSpec
	a := Alias(r)

	if len(r.DeployItems) > 0 {
		diBytes, err := json.Marshal(r.DeployItems)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal deployitems of execution spec: %w", err)
		}

		diZippedBytes, err := utils.Gzip(diBytes)
		if err != nil {
			return nil, fmt.Errorf("unable to gzip deployitems of execution spec: %w", err)
		}

		a.DeployItems = nil
		a.DeployItemsCompressed = diZippedBytes
	}

	return json.Marshal(a)
}
