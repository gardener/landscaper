// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsconfigv1alpha1 "github.com/gardener/landscaper/apis/config/v1alpha1"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration is the container deployer configuration that configures the controller
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	// Identity identity describes the unique identity of the deployer.
	// +optional
	Identity string `json:"identity,omitempty"`

	// OCI configures the oci client of the controller
	// +optional
	OCI *config.OCIConfiguration `json:"oci,omitempty"`

	// Namespace defines the namespace where the pods should be executed.
	// Defaults to default
	// +optional
	Namespace string `json:"namespace"`

	// TargetSelector describes all selectors the deployer should depend on.
	TargetSelector []lsv1alpha1.TargetSelector `json:"targetSelector,omitempty"`

	// DefaultImage configures the default images that is used if the DeployItem
	// does not specify one.
	DefaultImage ContainerSpec `json:"defaultImage"`

	// InitContainerImage defines the image that is used to init the container.
	// This container bootstraps the necessary directories and files.
	InitContainer ContainerSpec `json:"initContainer"`

	// SidecarContainerImage defines the image that is used as a
	// sidecar to the defined main container.
	// The sidecar container is responsible to collect the exports and the state of the main container.
	WaitContainer ContainerSpec `json:"waitContainer"`

	// GarbageCollection configures the container deployer garbage collector.
	GarbageCollection GarbageCollection `json:"garbageCollection"`

	// DebugOptions configure additional debug options.
	DebugOptions *DebugOptions `json:"debug,omitempty"`

	// Controller contains configuration concerning the controller framework.
	Controller Controller `json:"controller,omitempty"`
}

// ContainerSpec defines a container specification
type ContainerSpec struct {
	// Docker image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	// The image will be defaulted by the container deployer to the configured default.
	Image string `json:"image,omitempty"`
	// Entrypoint array. Not executed within a shell.
	// The docker image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Command []string `json:"command,omitempty"`
	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Args []string `json:"args,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// GarbageCollection defines the container deployer garbage collection configuration.
type GarbageCollection struct {
	// Disable disables the garbage collector and the resources clean-up.
	Disable bool `json:"disable"`
	// Worker defines the number of parallel garbage collection routines.
	// Defaults to 5.
	Worker int `json:"worker"`
	// RequeueTime specifies the duration after which the object, which is not yet ready to be garbage collected, is requeued.
	// Defaults to 60.
	RequeueTimeSeconds int `json:"requeueTimeSeconds"`
}

// DebugOptions defines optional debug options.
type DebugOptions struct {
	// KeepPod will only remove the finalizer on the pod but will not delete the pod.
	KeepPod bool `json:"keepPod,omitempty"`
}

// Controller contains configuration concerning the controller framework.
type Controller struct {
	lsconfigv1alpha1.CommonControllerConfig
}
