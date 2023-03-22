// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
)

// DeployItemType defines the type of the deploy item
type DeployItemType string

type DeployItemPhase string

type DeployerPhase string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeployItemList contains a list of DeployItems
type DeployItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeployItem `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeployItem defines a resource that should be processed by a external deployer
// +kubebuilder:subresource:status
type DeployItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeployItemSpec `json:"spec"`

	// +optional
	Status DeployItemStatus `json:"status"`
}

// DeployItemSpec contains the definition of a deploy item.
type DeployItemSpec struct {
	// Type is the type of the deployer that should handle the item.
	Type DeployItemType `json:"type"`
	// Target specifies an optional target of the deploy item.
	// In most cases it contains the secrets to access a evironment.
	// It is also used by the deployers to determine the ownernship.
	// +optional
	Target *ObjectReference `json:"target,omitempty"`
	// Context defines the current context of the deployitem.
	// +optional
	Context string `json:"context,omitempty"`
	// Configuration contains the deployer type specific configuration.
	Configuration *runtime.RawExtension `json:"config,omitempty"`
	// RegistryPullSecrets defines a list of registry credentials that are used to
	// pull blueprints, component descriptors and jsonschemas from the respective registry.
	// For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// Note that the type information is used to determine the secret key and the type of the secret.
	// +optional
	RegistryPullSecrets []ObjectReference `json:"registryPullSecrets,omitempty"`
	// Timeout specifies how long the deployer may take to apply the deploy item.
	// When the time is exceeded, the deploy item fails.
	// Value has to be parsable by time.ParseDuration (or 'none' to deactivate the timeout).
	// Defaults to ten minutes if not specified.
	// +optional
	Timeout *Duration `json:"timeout,omitempty"`
	// UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed.
	// +optional
	UpdateOnChangeOnly bool `json:"updateOnChangeOnly,omitempty"`
}

// DeployItemStatus contains the status of a deploy item
type DeployItemStatus struct {
	// Phase is the current phase of the DeployItem
	Phase DeployItemPhase `json:"phase,omitempty"`

	// ObservedGeneration is the most recent generation observed for this DeployItem.
	// It corresponds to the DeployItem generation, which is updated on mutation by the landscaper.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the actual condition of a deploy item
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`

	// LastError describes the last error that occurred.
	LastError *Error `json:"lastError,omitempty"`

	// ErrorHistory describes the last n errors that occurred since JobID was changed the last time.
	LastErrors []*Error `json:"lastErrors,omitempty"`

	// FirstError describes the first error that occurred since JobID was changed the last time.
	FirstError *Error `json:"firstError,omitempty"`

	// LastReconcileTime indicates when the reconciliation of the last change to the deploy item has started
	// +optional
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// Deployer describes the deployer that has reconciled the deploy item.
	// +optional
	Deployer DeployerInformation `json:"deployer,omitempty"`

	// ProviderStatus contains the provider specific status
	// +optional
	ProviderStatus *runtime.RawExtension `json:"providerStatus,omitempty"`

	// ExportReference is the reference to the object that contains the exported values.
	// +optional
	ExportReference *ObjectReference `json:"exportRef,omitempty"`

	// JobID is the ID of the current working request.
	JobID string `json:"jobID,omitempty"`

	// JobIDFinished is the ID of the finished working request.
	JobIDFinished string `json:"jobIDFinished,omitempty"`

	// JobIDGenerationTime is the timestamp when the JobID was set.
	JobIDGenerationTime *metav1.Time `json:"jobIDGenerationTime,omitempty"`

	// DeployerPhase is DEPRECATED and will soon be removed.
	DeployerPhase *string `json:"deployItemPhase,omitempty"`
}

// DeployerInformation holds additional information about the deployer that
// has reconciled or is reconciling the deploy item.
type DeployerInformation struct {
	// Identity describes the unique identity of the deployer.
	Identity string `json:"identity"`
	// Name is the name of the deployer.
	Name string `json:"name"`
	// Version is the version of the deployer.
	Version string `json:"version"`
}

// TargetSelector describes a selector that matches specific targets.
// +k8s:deepcopy-gen=true
type TargetSelector struct {
	// Targets defines a list of specific targets (name and namespace)
	// that should be reconciled.
	// +optional
	Targets []ObjectReference `json:"targets,omitempty"`
	// Annotations matches a target based on annotations.
	// +optional
	Annotations []Requirement `json:"annotations,omitempty"`
	// Labels matches a target based on its labels.
	// +optional
	Labels []Requirement `json:"labels,omitempty"`
}

// Requirement contains values, a key, and an operator that relates the key and values.
// The zero value of Requirement is invalid.
// Requirement implements both set based match and exact match
// Requirement should be initialized via NewRequirement constructor for creating a valid Requirement.
// +k8s:deepcopy-gen=true
type Requirement struct {
	Key      string             `json:"key"`
	Operator selection.Operator `json:"operator"`
	// In huge majority of cases we have at most one value here.
	// It is generally faster to operate on a single-element slice
	// than on a single-element map, so we have a slice here.
	// +optional
	Values []string `json:"values,omitempty"`
}
