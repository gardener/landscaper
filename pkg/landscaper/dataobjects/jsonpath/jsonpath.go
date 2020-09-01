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
	"errors"
	"fmt"
	"reflect"
	"strings"

	errors2 "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/yaml"
)

func GetValue(path string, data interface{}, out interface{}) error {
	outVal := reflect.ValueOf(out)
	if outVal.Kind() != reflect.Ptr {
		return errors.New("expected pointer")
	}

	if !strings.HasPrefix(path, ".") {
		path = "." + path
	}
	jp := jsonpath.New("get")
	if err := jp.Parse(fmt.Sprintf("{%s}", path)); err != nil {
		return err
	}

	res, err := jp.FindResults(data)
	if err != nil {
		return err
	}

	if len(res) == 0 {
		return errors.New("not found")
	}

	if len(res) != 1 && len(res[0]) != 1 {
		return errors.New("expected exactly one result")
	}
	val := reflect.Indirect(res[0][0])

	outVal = outVal.Elem()
	if !val.Type().AssignableTo(outVal.Type()) {
		errMsg := fmt.Sprintf("type %s is not assignable to type %s", val.Type(), outVal.Type())
		// if the value is of kind interface and the types are not assignable lets try to marshal and unmarshal the values
		if val.Kind() == reflect.Interface {
			data, err := yaml.Marshal(val.Interface())
			if err != nil {
				return errors2.Wrap(err, errMsg)
			}
			return yaml.Unmarshal(data, out)
		}

		return errors.New(errMsg)
	}

	outVal.Set(val)
	return nil
}

// Construct creates a map for the given jsonpath
// the value if the resulting map is set to the given value parameter
func Construct(text string, value interface{}) (map[string]interface{}, error) {
	if !strings.HasPrefix(text, ".") {
		text = "." + text
	}
	parser, err := jsonpath.Parse("construct", fmt.Sprintf("{%s}", text))
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{})
	if _, err := constructWalk(out, parser.Root, value); err != nil {
		return nil, err
	}
	return out, nil
}

func constructWalk(input map[string]interface{}, nodes *jsonpath.ListNode, value interface{}) (map[string]interface{}, error) {
	var (
		err     error
		fldPath = field.NewPath("")
	)
	curValue := input
	for i, node := range nodes.Nodes {
		switch n := node.(type) {
		case *jsonpath.ListNode:
			curValue, err = constructWalk(curValue, n, value)
			if err != nil {
				return curValue, err
			}
		case *jsonpath.FieldNode:
			newValue := make(map[string]interface{})
			fldPath = fldPath.Child(n.Value)
			curValue[n.Value] = newValue

			// if the node is the last in the list we can add the value
			if i == len(nodes.Nodes)-1 {
				curValue[n.Value] = value
				return curValue, nil
			}

			curValue = newValue
		default:
			return curValue, field.NotSupported(fldPath, node.Type(), []string{jsonpath.NodeTypeName[jsonpath.NodeList], jsonpath.NodeTypeName[jsonpath.NodeField]})
		}
	}

	return curValue, nil
}
