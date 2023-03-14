// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package managedresource

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ManifestPolicy defines the strategy how a mnifest should be managed
// by the deployer.
type ManifestPolicy string

const (
	// ManagePolicy is the default policy where the resource is
	// created, updated and deleted and it occupies already managed resources
	ManagePolicy ManifestPolicy = "manage"
	// FallbackPolicy defines a policy where the resource is created, updated and deleted
	// but only if not already managed by someone else (check for annotation with landscaper identity, deployitem name + namespace)
	FallbackPolicy ManifestPolicy = "fallback"
	// KeepPolicy defines a policy where the resource is only created and updated but not deleted.
	// It is not deleted when the whole deploy item is nor when the resource is not defined anymore.
	KeepPolicy ManifestPolicy = "keep"
	// IgnorePolicy defines a policy where the resource is completely ignored by the deployer.
	IgnorePolicy ManifestPolicy = "ignore"
	// ImmutablePolicy defines a policy where the resource is created and deleted but never updated.
	ImmutablePolicy ManifestPolicy = "immutable"
)

// Manifest defines a manifest that is managed by the deployer.
type Manifest struct {
	// Policy defines the manage policy for that resource.
	Policy ManifestPolicy `json:"policy,omitempty"`
	// Manifest defines the raw k8s manifest.
	Manifest *runtime.RawExtension `json:"manifest,omitempty"`
	// AnnotateBeforeCreate defines annotations that are being set before the manifest is being created.
	// +optional
	AnnotateBeforeCreate map[string]string `json:"annotateBeforeCreate,omitempty"`
	// AnnotateBeforeDelete defines annotations that are being set before the manifest is being deleted.
	// +optional
	AnnotateBeforeDelete map[string]string `json:"annotateBeforeDelete,omitempty"`
}

// ManagedResourceStatusList describes a list of managed resource statuses.
type ManagedResourceStatusList []ManagedResourceStatus

// ObjectReferenceList converts a ManagedResourceStatusList to a list of typed objet references.
func (mr ManagedResourceStatusList) ObjectReferenceList() []corev1.ObjectReference {
	list := make([]corev1.ObjectReference, len(mr))
	for i, res := range mr {
		list[i] = res.Resource
	}
	return list
}

// TypedObjectReferenceList converts a ManagedResourceStatusList to a list of typed objet references.
func (mr ManagedResourceStatusList) TypedObjectReferenceList() []lsv1alpha1.TypedObjectReference {
	list := make([]lsv1alpha1.TypedObjectReference, len(mr))
	for i, res := range mr {
		list[i] = lsv1alpha1.TypedObjectReference{
			APIVersion: res.Resource.APIVersion,
			Kind:       res.Resource.Kind,
			ObjectReference: lsv1alpha1.ObjectReference{
				Name:      res.Resource.Name,
				Namespace: res.Resource.Namespace,
			},
		}
	}
	return list
}

// ManagedResourceStatus describes the managed resource and their metadata.
type ManagedResourceStatus struct {
	// AnnotateBeforeDelete defines annotations that are being set before the manifest is being deleted.
	// +optional
	AnnotateBeforeDelete map[string]string `json:"annotateBeforeDelete,omitempty"`
	// Policy defines the manage policy for that resource.
	Policy ManifestPolicy `json:"policy,omitempty"`
	// Resources describes the managed kubernetes resource.
	Resource corev1.ObjectReference `json:"resource"`
}

// Exports describes one export that is read from a resource.
type Exports struct {
	// DefaultTimeout defines the default timeout for all exports
	// that the exporter waits for the value in the jsonpath to occur.
	DefaultTimeout *lsv1alpha1.Duration `json:"defaultTimeout,omitempty"`
	Exports        []Export             `json:"exports,omitempty"`
}

// Export describes one export that is read from a resource.
type Export struct {
	// Timeout defines the timeout that the exporter waits for the value in the jsonpath to occur.
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`
	// Key is the key that the value from JSONPath is exported to.
	Key string `json:"key"`

	// JSONPath is the jsonpath to look for a value.
	// The JSONPath root is the referenced resource
	JSONPath string `json:"jsonPath"`

	// FromResource specifies the name of the resource where the value should be read.
	FromResource *lsv1alpha1.TypedObjectReference `json:"fromResource,omitempty"`

	// FromObjectReference describes that the jsonpath points to a object reference where the actual value is read from.
	// This is helpful if for example a deployed resource referenced a secret and that exported value is in that secret.
	FromObjectReference *FromObjectReference `json:"fromObjectRef,omitempty"`
}

// FromObjectReference describes that the jsonpath points to a object reference where the actual value is read from.
// This is helpful if for example a deployed resource referenced a secret and that exported value is in that secret.
type FromObjectReference struct {
	// APIVersion is the group and version for the resource being referenced.
	// If APIVersion is not specified, the specified Kind must be in the core API group.
	// For any other third-party types, APIVersion is required.
	APIVersion string `json:"apiVersion"`
	// Kind is the type of resource being referenced
	Kind string `json:"kind"`
	// JSONPath is the jsonpath to look for a value.
	// The JSONPath root is the referenced resource
	JSONPath string `json:"jsonPath"`
}
