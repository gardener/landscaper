// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

var Any = map[string]interface{}{"x-kubernetes-preserve-unknown-fields": true}

func replaceTypesInStruct(data map[string]interface{}) map[string]interface{} {
	for key, val := range data {
		switch v := val.(type) {
		case map[string]interface{}:
			if containsTypeAny(v) {
				data[key] = Any
				continue
			}
			if isAny("items", key, v) || isAny("additionalProperties", key, v) {
				data[key] = Any
				continue
			}
			data[key] = replaceTypesInStruct(v)
		case []interface{}:
			data[key] = replaceInArray(v)
		default:
			continue
		}
	}
	return data
}

func replaceInArray(data []interface{}) []interface{} {
	updated := make([]interface{}, len(data))
	for i, val := range data {
		switch v := val.(type) {
		case map[string]interface{}:
			updated[i] = replaceTypesInStruct(v)
		case []interface{}:
			updated[i] = replaceInArray(v)
			continue
		default:
			updated[i] = val
		}
	}
	return updated
}

func containsTypeAny(data map[string]interface{}) bool {
	val, ok := data["type"]
	if !ok {
		return false
	}
	if s, ok := val.(string); ok && s == "Any" {
		return true
	}
	return false
}

func isAny(expected, key string, val interface{}) bool {
	if key != expected {
		return false
	}
	if s, ok := val.(map[string]interface{}); ok && len(s) == 0 {
		return true
	}
	return false
}
