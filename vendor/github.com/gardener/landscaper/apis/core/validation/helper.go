// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
)

var importTypesWithExpectedConfig = map[string][]string{
	string(core.ImportTypeData):       {"Schema"},
	string(core.ImportTypeTarget):     {"TargetType"},
	string(core.ImportTypeTargetList): {"TargetType"},
}
var exportTypesWithExpectedConfig = map[string][]string{
	string(core.ExportTypeData):   {"Schema"},
	string(core.ExportTypeTarget): {"TargetType"},
}

var relevantConfigFields map[string]bool

// isFieldValueDefinition lists the fields which are not directly part of an Import/ExportDefinition, but of the FieldValueDefinition
// (usually the fields which are present in both import and export definitions)
// This is required for the reflection used below
// a map is used for easier 'contains' queries, the values are ignored
var isFieldValueDefinition = map[string]bool{
	"Schema":     true,
	"TargetType": true,
}

// keys returns all keys of a map as a slice
func keys(m map[string][]string) []string {
	res := make([]string, len(m))
	for k := range m {
		res = append(res, string(k))
	}
	return res
}

// computeRelevantConfigFields returns the union of the values of importTypesWithExpectedConfig and exportTypesWithExpectedConfig
// as a map[string]bool for easier 'contains' queries. The value will always be set to 'true'.
// This is used to detect if a config field is set which should not be set for the specified type.
func computeRelevantConfigFields() map[string]bool {
	res := map[string]bool{}
	for _, v := range importTypesWithExpectedConfig {
		for _, e := range v {
			res[e] = true
		}
	}
	for _, v := range exportTypesWithExpectedConfig {
		for _, e := range v {
			res[e] = true
		}
	}
	return res
}

// stringContains is a small helper function that checks whether a string is contained in a string slice
func stringContains(data []string, value string) bool {
	for _, elem := range data {
		if elem == value {
			return true
		}
	}
	return false
}

// ValidateExactlyOneOf is a helper function that takes a struct and a list of field names and validates that exactly one of
// the specified fields has a non-nil, non-zero value. If that's the case, an empty ErrorList will be returned.
// If the given struct is a nil pointer, this will be treated as all fields being nil/zero and return an error.
func ValidateExactlyOneOf(fldPath *field.Path, input interface{}, configs ...string) field.ErrorList {
	setFields := []string{}
	val := reflect.ValueOf(input)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return field.ErrorList{field.Required(fldPath, fmt.Sprintf("exactly one of [%s] must be set (currently set: none)", strings.Join(configs, ", ")))}
		}
		val = reflect.Indirect(val)
	}
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		fieldName := val.Type().Field(i).Name
		kind := f.Kind()
		if !stringContains(configs, fieldName) {
			// field is not relevant
			continue
		}
		// check if field is set
		if ((kind == reflect.Ptr || kind == reflect.Slice || kind == reflect.Map || kind == reflect.Interface) && !f.IsNil()) || !f.IsZero() {
			// field is set
			setFields = append(setFields, fieldName)
		}
	}

	if len(setFields) > 1 {
		return field.ErrorList{field.Invalid(fldPath, input, fmt.Sprintf("exactly one of [%s] must be set (currently set: [%s])", strings.Join(configs, ", "), strings.Join(setFields, ", ")))}
	} else if len(setFields) == 0 {
		return field.ErrorList{field.Required(fldPath, fmt.Sprintf("exactly one of [%s] must be set (currently set: none)", strings.Join(configs, ", ")))}
	}
	return field.ErrorList{}
}
