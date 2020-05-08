// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"encoding/json"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExecutionType defines the type of the execution
type ExecutionType string

const (
	// ExecutionTypeContainer defines the container executor
	ExecutionTypeContainer ExecutionType = "container"

	// ExecutionTypeScript defines the script executor
	ExecutionTypeScript ExecutionType = "script"

	// ExecutionTypeTemplate defines the template executor
	ExecutionTypeTemplate ExecutionType = "template"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentDefinitionList contains a list of Definitions
type ComponentDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentDefinition `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentDefinition contains the configuration of a component
// +kubebuilder:subresource:status
type ComponentDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DefinitionSpec `json:"spec"`

	Status DefinitionStatus `json:"status"`
}

type DefinitionSpec struct {
	CustomTypes []CustomType `json:"customTypes,omitempty"`

	Import json.RawMessage `json:"import,omitempty"`
	Export json.RawMessage `json:"import,omitempty"`

	// Executors defines the executors that are sequentially executed by the landscaper
	Executors []Execution `json:"executors"`
}

type DefinitionStatus struct {
	// ObservedGeneration is the most recent generation observed for this Type. It corresponds to the
	// Shoot's generation, which is updated on mutation by the landscaper.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the last observed conditions of the component definition.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

type CustomType struct {
	Name            string                          `json:"name"`
	OpenAPIV3Schema apiextensionsv1.JSONSchemaProps `json:"openAPIV3Schema,omitempty"`
}

type Execution struct {
	Type ExecutionType `json:"type"`

	// +optional
	ContainerConfig *ContainerConfig `json:"containerConfig,omitempty"`

	// +optional
	ScriptConfig    *ScriptConfig    `json:"scriptConfig,omitempty"`
}

type ContainerConfig struct {
	// Docker image name.
	// +optional
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
	// Container's working directory.
	// If not specified, the container runtime's default will be used, which
	// might be configured in the container image.
	// Cannot be updated.
	// +optional
	WorkingDir string `json:"workingDir,omitempty"`
}

type ScriptConfig struct {
	Script string `json:"script"`
	// Docker image name.
	// +optional
	Image string `json:"image,omitempty"`
}

type TemplateConfig struct {
	Config string `json:"config"`
}
