// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"encoding/json"
	"errors"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// JSONSchemaDefinition defines a jsonschema.
type JSONSchemaDefinition json.RawMessage

// MarshalJSON implements the json marshaling for a JSON
func (s JSONSchemaDefinition) MarshalJSON() ([]byte, error) {
	return json.RawMessage(s).MarshalJSON()
}

// UnmarshalJSON implements json unmarshaling for a JSON
func (s *JSONSchemaDefinition) UnmarshalJSON(data []byte) error {
	raw := json.RawMessage{}
	if err := raw.UnmarshalJSON(data); err != nil {
		return err
	}
	*s = JSONSchemaDefinition(raw)
	return nil
}

// ConditionStatus is the status of a condition.
type ConditionStatus string

// ConditionType is a string alias.
type ConditionType string

const (
	// ConditionTrue means a resource is in the condition.
	ConditionTrue ConditionStatus = "True"
	// ConditionFalse means a resource is not in the condition.
	ConditionFalse ConditionStatus = "False"
	// ConditionUnknown means Landscaper can't decide if a resource is in the condition or not.
	ConditionUnknown ConditionStatus = "Unknown"
	// ConditionProgressing means the condition was seen true, failed but stayed within a predefined failure threshold.
	// In the future, we could add other intermediate conditions, e.g. ConditionDegraded.
	ConditionProgressing ConditionStatus = "Progressing"

	// ConditionCheckError is a constant for a reason in condition.
	ConditionCheckError = "ConditionCheckError"
)

// ErrorCode is a string alias.
type ErrorCode string

const (
	// ErrorUnauthorized indicates that the last error occurred due to invalid credentials.
	ErrorUnauthorized ErrorCode = "ERR_UNAUTHORIZED"
	// ErrorCleanupResources indicates that the last error occurred due to resources are stuck in deletion.
	ErrorCleanupResources ErrorCode = "ERR_CLEANUP"
	// ErrorConfigurationProblem indicates that the last error occurred due a configuration problem.
	ErrorConfigurationProblem ErrorCode = "ERR_CONFIGURATION_PROBLEM"
)

// Condition holds the information about the state of a resource.
type Condition struct {
	// DataType of the Shoot condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Last time the condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message"`
	// Well-defined error codes in case the condition reports a problem.
	// +optional
	Codes []ErrorCode `json:"codes,omitempty"`
}

// Error holds information about an error that occurred.
type Error struct {
	// Operation describes the operator where the error ocurred.
	Operation string `json:"operation"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Last time the condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message"`
	// Well-defined error codes in case the condition reports a problem.
	// +optional
	Codes []ErrorCode `json:"codes,omitempty"`
}

type Operation string

const (
	// ReconcileOperation is a annotation for the landscaper to reconcile the resource
	ReconcileOperation Operation = "reconcile"

	// ForceReconcileOperation forces the landscaper to not wait for children (executions nor subinstallations) to be completed.
	ForceReconcileOperation Operation = "force-reconcile"

	// AbortOperation is the annotation to let the landscaper abort all currently running children and itself.
	AbortOperation Operation = "abort"
)

// ObjectReference is the reference to a kubernetes object.
type ObjectReference struct {
	// Name is the name of the kubernetes object.
	Name string `json:"name"`

	// Namespace is the namespace of kubernetes object.
	Namespace string `json:"namespace"`
}

// NamespacedName returns the namespaced name for the object reference
func (r *ObjectReference) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      r.Name,
		Namespace: r.Namespace,
	}
}

// TypedObjectReference is a reference to a typed kubernetes object.
type TypedObjectReference struct {
	// APIVersion is the group and version for the resource being referenced.
	// If APIVersion is not specified, the specified Kind must be in the core API group.
	// For any other third-party types, APIVersion is required.
	APIVersion string `json:"apiVersion"`
	// Kind is the type of resource being referenced
	Kind string `json:"kind"`

	ObjectReference `json:",inline"`
}

// NamedObjectReference is a named reference to a specific resource.
type NamedObjectReference struct {
	// Name is the unique name of the reference.
	Name string `json:"name"`

	// Reference is the reference to an object.
	Reference ObjectReference `json:"ref"`
}

// VersionedObjectReference is a reference to a object with its last observed resource generation.
// This struct is used by status fields.
type VersionedObjectReference struct {
	ObjectReference `json:",inline"`

	// ObservedGeneration defines the last observed generation of the referenced resource.
	ObservedGeneration int64 `json:"observedGeneration"`
}

// VersionedObjectReference is a named reference to a object with its last observed resource generation.
// This struct is used by status fields.
type VersionedNamedObjectReference struct {
	// Name is the unique name of the reference.
	Name string `json:"name"`

	// Reference is the reference to an object.
	Reference VersionedObjectReference `json:"ref"`
}

// ResourceKind is the kind of a resource.
// It can be local or external.
type ResourceKind string

var UnknownResourceKindError = errors.New("UnknownResourceKind")

const (
	// LocalResourceKind is the kind of a local resource.
	LocalResourceKind ResourceKind = "localResource"
	// ExternalResourceKind is the kind of a external resource.
	ExternalResourceKind ResourceKind = "externalResource"
)

// ResourceReference defines the reference to a resource defined in a component descriptor.
type ResourceReference struct {
	// ComponentName defines the unique of the component containing the resource.
	ComponentName string `json:"componentName"`
	// Kind defines the kind resource kind where the resource is defined.
	Kind ResourceKind `json:"kind"`
	// ResourceName defines the name of the resource.
	ResourceName string `json:"resourceName"`
}

// VersionedResourceReference defines the reference to a resource with its version.
type VersionedResourceReference struct {
	ResourceReference `json:",inline"`
	// Version defines the version of the component.
	Version string `json:"version"`
}

// ObjectMeta returns the component descriptor v2 compatible object meta for a resource reference.
func (r VersionedResourceReference) ObjectMeta() cdv2.ObjectMeta {
	return cdv2.ObjectMeta{
		Name:    r.ComponentName,
		Version: r.Version,
	}
}
