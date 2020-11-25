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

package codec

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/ghodss/yaml"

	"github.com/gardener/component-spec/bindings-go/apis"
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/apis/v2/jsonscheme"
	"github.com/gardener/component-spec/bindings-go/apis/v2/validation"
)

type serializer struct {
	knownAccessTypes v2.KnownTypes
}

// Decode decodes a component into the given object.
// The obj is expected to be of type v2.ComponentDescriptor or v2.ComponentDescriptorList.
func Decode(data []byte, obj interface{}) error {
	objType := reflect.TypeOf(obj)
	if objType.Kind() != reflect.Ptr {
		return fmt.Errorf("object is expected to be of type pointer but is of type %T", obj)
	}

	raw := make(map[string]json.RawMessage)
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}

	var metadata apis.Metadata
	if err := yaml.Unmarshal(raw["meta"], &metadata); err != nil {
		return err
	}

	// handle v2
	if metadata.Version == v2.SchemaVersion && objType.Elem() == reflect.TypeOf(v2.ComponentDescriptor{}) {
		if err := jsonscheme.Validate(data); err != nil {
			return err
		}
		if err := yaml.Unmarshal(data, obj); err != nil {
			return err
		}
		comp := obj.(*v2.ComponentDescriptor)
		if err := v2.DefaultComponent(comp); err != nil {
			return err
		}
		return validation.Validate(comp)
	}

	if metadata.Version == v2.SchemaVersion && objType.Elem() == reflect.TypeOf(v2.ComponentDescriptorList{}) {
		if err := yaml.Unmarshal(data, obj); err != nil {
			return err
		}
		list := obj.(*v2.ComponentDescriptorList)
		if err := v2.DefaultList(list); err != nil {
			return err
		}
		return validation.ValidateList(list)
	}

	// todo: implement conversion
	return errors.New("invalid version")
}

// Encode encodes a component or component list into the given object.
// The obj is expected to be of type v2.ComponentDescriptor or v2.ComponentDescriptorList.
func Encode(obj interface{}) ([]byte, error) {
	objType := reflect.TypeOf(obj)
	if objType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("object is expected to be of type pointer but is of type %T", obj)
	}

	if objType.Elem() == reflect.TypeOf(v2.ComponentDescriptor{}) {
		comp := obj.(*v2.ComponentDescriptor)
		comp.Metadata.Version = v2.SchemaVersion
		if err := v2.DefaultComponent(comp); err != nil {
			return nil, err
		}
		return json.Marshal(comp)
	}

	if objType.Elem() == reflect.TypeOf(v2.ComponentDescriptorList{}) {
		list := obj.(*v2.ComponentDescriptorList)
		list.Metadata.Version = v2.SchemaVersion
		if err := v2.DefaultList(list); err != nil {
			return nil, err
		}
		return json.Marshal(list)
	}

	// todo: implement conversion
	return nil, errors.New("invalid version")
}
