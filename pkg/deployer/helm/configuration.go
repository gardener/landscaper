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
	"errors"

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

	ExportedFiles []string
}

// Status is the helm provider specific status
type Status struct {
	ManagedResources []lsv1alpha1.TypedObjectReference `json:"managedResources,omitempty"`
}

// Validate validates a configuration object
func Validate(config *Configuration) error {
	if len(config.Repository) == 0 {
		return errors.New("respoitory has to be defined")
	}
	if len(config.Version) == 0 {
		return errors.New("version has to be defined")
	}
	return nil
}
