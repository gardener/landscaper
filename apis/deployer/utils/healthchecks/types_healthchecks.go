// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package healthchecks

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// HealthChecksConfiguration contains the configuration for health checks.
type HealthChecksConfiguration struct {
	// DisableDefault allows to disable the default health checks.
	// +optional
	DisableDefault bool `json:"disableDefault,omitempty"`
	// Timeout is the time to wait before giving up on a resource to be healthy.
	// Defaults to 180s.
	// +optional
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`
	// CustomHealthChecks is a set of custom health check configurations
	// +optional
	CustomHealthChecks []CustomHealthCheckConfiguration `json:"custom,omitempty"`
}

// CustomHealthCheckConfiguration contains the configuration for a custom health check
type CustomHealthCheckConfiguration struct {
	// Name is the name of the HealthCheck
	Name string `json:"name"`
	// Timeout is the value after which a HealthCheck should time out
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`
	// Disabled deactivates this custom resource health check
	// +optional
	Disabled bool `json:"disabled,omitempty"`
	// Resource is the resource for which the health-check should be applied, used for single resources that can be identified by namespace and name
	// +optional
	Resource []lsv1alpha1.TypedObjectReference `json:"resourceSelector,omitempty"`
	// Labels are the labels used to identify multiple resources that can be identified by a unique set of labels
	// +optional
	LabelSelector *LabelSelectorSpec `json:"labelSelector,omitempty"`
	// Requirements is the actual health-check which compares an object's property to a value
	Requirements []RequirementSpec `json:"requirements"`
}

// LabelSelectorSpec contains paramters used to select objects by their labels
type LabelSelectorSpec struct {
	// APIVersion is the API version of the object to be selected by labels
	APIVersion string `json:"apiVersion"`
	// Kind is the Kind of the object to be selected by labels
	Kind string `json:"kind"`
	// Labels are the labels used to identify multiple resources of the given kind
	Labels map[string]string `json:"matchLabels"`
}

// RequirementSpec contains the requirements an object must meet to pass the custom health check
type RequirementSpec struct {
	JsonPath string             `json:"jsonPath"`
	Operator selection.Operator `json:"operator"`
	// In huge majority of cases we have at most one value here.
	// It is generally faster to operate on a single-element slice
	// than on a single-element map, so we have a slice here.
	// +optional
	Value []runtime.RawExtension `json:"values,omitempty"`
}
