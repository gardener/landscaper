// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Values defines a local values file that conatins the imports to be used to render the blueprint.
type Values struct {
	Imports map[string]interface{} `json:"imports,omitempty"`
}

// ValidateValues validates values file.
func ValidateValues(values *Values) error {
	allErrs := field.ErrorList{}

	fldPath := field.NewPath("imports")
	for key, val := range values.Imports {
		if val == nil {
			allErrs = append(allErrs, field.Required(fldPath.Key(key), "value must not be empty"))
		}
	}

	return allErrs.ToAggregate()
}

// MergeValues merges all values of b into a.
func MergeValues(a, b *Values) {
	if a.Imports == nil {
		a.Imports = b.Imports
		return
	}
	for key, val := range b.Imports {
		a.Imports[key] = val
	}
}
