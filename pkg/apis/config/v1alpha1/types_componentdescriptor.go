// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentList is a list of components
type ComponentList struct {
	metav1.TypeMeta `json:",inline"`
	Components      []*Component `json:"components"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Component represents a arbitrary versioned component with artifacts and dependencies to other components
type Component struct {
	DependencyMeta `json:",inline"`

	// Name is the unique name of a component.
	Name string `json:"name"`

	// Version is the version of the component.
	Version string `json:"version"`

	// Dependencies contains all dependencies of various types
	Dependencies Dependencies `json:"dependencies"`
}

// ComponentOverwrite defines an overwrite for component artifacts
type ComponentOverwrite struct {
	// DeclaringComponent is the name of the component to be overwritten
	DeclaringComponent DependencyMeta `json:"declaring_component"`

	// DependencyOverwrites declares dependency overwrites for the component
	DependencyOverwrites Dependencies `json:"dependency_overwrites"`
}

// Dependencies describes a dependency of a Component.
// The dependency can be of various types like another component or a imgage, etc..
type Dependencies struct {
	// Components contains all dependencies to other components
	Components []ComponentDependency `json:"components"`

	// ContainerImages contains all container image dependencies.
	ContainerImages []ImageDependency `json:"container_images"`

	// HelmCharts contains all helm chart dependencies.
	HelmCharts []HelmChartDependency `json:"helm_charts"`
}

// DependencyMeta is the metadata that should be included in all dependency types.
type DependencyMeta struct {
	// Name is the unique name of a dependency.
	Name string `json:"name"`

	// Version is the version of the dependency.
	Version string `json:"version"`
}

// ComponentDependency describes a dependency to another component.
type ComponentDependency struct {
	DependencyMeta `json:",inline"`
}

// ImageDependency describes a dependency to a image.
type ImageDependency struct {
	DependencyMeta `json:",inline"`

	// Reference is the oci url to a image.
	Reference string `json:"image_reference"`
}

// HelmChartDependency describes a dependency to a helm chart.
type HelmChartDependency struct {
	DependencyMeta `json:",inline"`

	// Reference is the oci url to the chart.
	Reference string `json:"chart_reference"`
}
