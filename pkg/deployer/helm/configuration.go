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

package helm

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Configuration is the configuration of a helm deploy item.
// todo: use versioned configuration
type Configuration struct {
	// Kubeconfig is the base64 encoded kubeconfig file
	Kubeconfig string `json:"kubeconfig"`

	// Repository is the repository name in the oci registry
	Repository string `json:"repository"`

	// Version is the chart version in the oci registry.
	Version string `json:"version"`

	// Name is the release name of the chart
	Name string `json:"name"`

	// Namespace is the release namespace of the chart
	Namespace string `json:"namespace"`

	// Values are the values that are used for templating.
	Values map[string]interface{} `json:"values,omitempty"`

	// ExportsFromManifests describe the exports from the templated manifests that should be exported by the helm deployer.
	// +optional
	ExportsFromManifests []ExportFromManifestItem `json:"exportsFromManifests,omitempty"`
}

// ExportFromManifestItem describes one export that is read from a templated resource.
type ExportFromManifestItem struct {
	// Key is the key that the value from JSONPath is exported to.
	Key string `json:"key"`

	// JSONPath is the jsonpath to look for a value.
	// The JSONPath root is the referenced resource
	JSONPath string `json:"jsonPath"`

	// Resource specifies the name of the resource where the value should be read.
	Resource lsv1alpha1.TypedObjectReference `json:"resource"`
}

// Status is the helm provider specific status
type Status struct {
	ManagedResources []lsv1alpha1.TypedObjectReference `json:"managedResources,omitempty"`
}

// Validate validates a configuration object
func Validate(config *Configuration) error {
	allErrs := field.ErrorList{}
	if len(config.Repository) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("repository"), "must not be empty"))
	}
	if len(config.Version) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("version"), "must not be empty"))
	}

	expPath := field.NewPath("exportsFromManifests")
	for i, export := range config.ExportsFromManifests {
		indexFldPath := expPath.Index(i)
		if len(export.Key) == 0 {
			allErrs = append(allErrs, field.Required(indexFldPath.Child("key"), "must not be empty"))
		}
		if len(export.JSONPath) == 0 {
			allErrs = append(allErrs, field.Required(indexFldPath.Child("jsonPath"), "must not be empty"))
		}

		resFldPath := indexFldPath.Child("resource")
		if len(export.Resource.APIVersion) == 0 {
			allErrs = append(allErrs, field.Required(resFldPath.Child("apiGroup"), "must not be empty"))
		}
		if len(export.Resource.Kind) == 0 {
			allErrs = append(allErrs, field.Required(resFldPath.Child("kind"), "must not be empty"))
		}
		if len(export.Resource.Name) == 0 {
			allErrs = append(allErrs, field.Required(resFldPath.Child("name"), "must not be empty"))
		}
		if len(export.Resource.Namespace) == 0 {
			allErrs = append(allErrs, field.Required(resFldPath.Child("namespace"), "must not be empty"))
		}
	}

	return allErrs.ToAggregate()
}
