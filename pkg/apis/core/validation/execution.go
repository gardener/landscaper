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

package validation

import (
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/apis/core"
)

// ValidateInstallation validates an Installation
func ValidateExecution(execution *core.Execution) error {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateExecutionSpec(field.NewPath("spec"), execution.Spec)...)
	return allErrs.ToAggregate()
}

// ValidateExecutionSpec validtes the spec of a execution object
func ValidateExecutionSpec(fldpath *field.Path, spec core.ExecutionSpec) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateDeployItemTemplateList(fldpath.Child("deployItems"), spec.DeployItems)...)
	return allErrs
}

// ValidateDeployItemTemplateList validates a list of deploy item templates.
func ValidateDeployItemTemplateList(fldPath *field.Path, list core.DeployItemTemplateList) field.ErrorList {
	allErrs := field.ErrorList{}
	names := sets.NewString()
	for i, tmpl := range list {
		tmplPath := fldPath.Index(i)
		if len(tmpl.Name) != 0 {
			if names.Has(tmpl.Name) {
				allErrs = append(allErrs, field.Duplicate(tmplPath, tmpl.Name))
			}
			names.Insert(tmpl.Name)
			tmplPath = tmplPath.Key(tmpl.Name)
		}
		allErrs = append(allErrs, ValidateDeployItemTemplate(tmplPath, tmpl)...)
	}

	return allErrs
}

// ValidateDeployItemTemplate validates a deploy item template.
func ValidateDeployItemTemplate(fldPath *field.Path, tmpl core.DeployItemTemplate) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(tmpl.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must not be empty"))
	}

	if len(tmpl.Type) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "type must not be empty"))
	}

	if tmpl.Target != nil {
		allErrs = append(allErrs, ValidateObjectReference(*tmpl.Target, fldPath.Child("target"))...)
	}

	if len(tmpl.Labels) != 0 {
		allErrs = append(allErrs, metav1validation.ValidateLabels(tmpl.Labels, fldPath.Child("labels"))...)
	}

	return allErrs
}
