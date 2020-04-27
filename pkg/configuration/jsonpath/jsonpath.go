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

package jsonpath

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/util/jsonpath"
)

// Construct creates a map for the given jsonpath
func Construct(text string) (map[string]interface{}, error) {
	parser, err := jsonpath.Parse("construct", text)
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{})
	if _, err := constructWalk(out, parser.Root); err != nil {
		return nil, err
	}
	return out, nil
}

func constructWalk(input map[string]interface{}, nodes *jsonpath.ListNode) (map[string]interface{}, error) {
	var (
		err     error
		fldPath = field.NewPath("")
	)
	curValue := input
	for _, node := range nodes.Nodes {
		switch n := node.(type) {
		case *jsonpath.ListNode:
			curValue, err = constructWalk(curValue, n)
			if err != nil {
				return curValue, err
			}
		case *jsonpath.FieldNode:
			newValue := make(map[string]interface{}, 0)
			fldPath = fldPath.Child(n.Value)
			curValue[n.Value] = newValue
			curValue = newValue
		default:
			return curValue, field.NotSupported(fldPath, node.Type(), []string{jsonpath.NodeTypeName[jsonpath.NodeList], jsonpath.NodeTypeName[jsonpath.NodeField]})
		}
	}
	return curValue, nil
}
