// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsconfigv1alpha1 "github.com/gardener/landscaper/apis/config/v1alpha1"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ManagedInstanceLabel describes label that is added to every helm deployer managed resource
// to define its corresponding instance.
const ManagedInstanceLabel = "helm.deployer.landscaper.gardener.cloud/instance"

// ManagedDeployItemLabel describes label that is added to every helm deployer managed resource
// to define its source deploy item.
const ManagedDeployItemLabel = "helm.deployer.landscaper.gardener.cloud/deployitem"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration is the helm deployer configuration that configures the controller
type Configuration struct {
	metav1.TypeMeta `json:",inline"`
	// Identity identity describes the unique identity of the deployer.
	// +optional
	Identity string `json:"identity,omitempty"`
	// OCI configures the oci client of the controller
	OCI *config.OCIConfiguration `json:"oci,omitempty"`
	// TargetSelector describes all selectors the deployer should depend on.
	TargetSelector []lsv1alpha1.TargetSelector `json:"targetSelector,omitempty"`
	// Export defines the export configuration.
	Export ExportConfiguration `json:"export,omitempty"`
	// HPAConfiguration contains the configuration for horizontal pod autoscaling.
	HPAConfiguration *HPAConfiguration `json:"hpa,omitempty"`
	// Controller contains configuration concerning the controller framework.
	Controller Controller `json:"controller,omitempty"`
	// +optional
	UseOCMLib bool `json:"useOCMLib,omitempty"`
}

// ExportConfiguration defines the export configuration for the deployer.
type ExportConfiguration struct {
	// DefaultTimeout configures the default timeout for all exports without a explicit export timeout defined.
	// +optional
	DefaultTimeout *lsv1alpha1.Duration `json:"defaultTimeout,omitempty"`
}

// HPAConfiguration contains the configuration for horizontal pod autoscaling.
type HPAConfiguration struct {
	MaxReplicas int32 `json:"maxReplicas,omitempty"`
}

// Controller contains configuration concerning the controller framework.
type Controller struct {
	lsconfigv1alpha1.CommonControllerConfig
}
