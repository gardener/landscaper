// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import "github.com/gardener/landscaper/apis/core"

var importTypesWithExpectedConfig = map[string][]string{
	string(core.ImportTypeData):   {"Schema"},
	string(core.ImportTypeTarget): {"TargetType"},
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
