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

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
)

// ValidateProviderConfiguration validates a helm deployer configuration
func ValidateProviderConfiguration(config *helmv1alpha1.ProviderConfiguration) error {
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
