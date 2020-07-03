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
