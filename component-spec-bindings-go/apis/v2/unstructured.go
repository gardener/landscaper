// Copyright 2021 Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package v2

import (
	"bytes"
	"encoding/json"
)

// UnstructuredTypesEqual compares two unstructured object.
func UnstructuredTypesEqual(a, b *UnstructuredTypedObject) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.GetType() != b.GetType() {
		return false
	}
	rawA, err := a.GetRaw()
	if err != nil {
		return false
	}
	rawB, err := b.GetRaw()
	if err != nil {
		return false
	}
	return bytes.Equal(rawA, rawB)
}

// TypedObjectEqual compares two typed objects using the unstructured type.
func TypedObjectEqual(a, b TypedObjectAccessor) bool {
	if a.GetType() != b.GetType() {
		return false
	}
	uA, err := NewUnstructured(a)
	if err != nil {
		return false
	}
	uB, err := NewUnstructured(b)
	if err != nil {
		return false
	}
	return UnstructuredTypesEqual(&uA, &uB)
}

// NewUnstructured creates a new unstructured object from a typed object using the default codec.
func NewUnstructured(obj TypedObjectAccessor) (UnstructuredTypedObject, error) {
	uObj, err := ToUnstructuredTypedObject(NewDefaultCodec(), obj)
	if err != nil {
		return UnstructuredTypedObject{}, nil
	}
	return *uObj, nil
}

// NewEmptyUnstructured creates a new typed object without additional data.
func NewEmptyUnstructured(ttype string) *UnstructuredTypedObject {
	return NewUnstructuredType(ttype, nil)
}

// NewUnstructuredType creates a new unstructured typed object.
func NewUnstructuredType(ttype string, data map[string]interface{}) *UnstructuredTypedObject {
	unstr := &UnstructuredTypedObject{}
	unstr.Object = data
	unstr.SetType(ttype)
	return unstr
}

// UnstructuredTypedObject describes a generic typed object.
// +k8s:openapi-gen=true
type UnstructuredTypedObject struct {
	ObjectType `json:",inline"`
	Raw        []byte                 `json:"-"`
	Object     map[string]interface{} `json:"-"`
}

func (u *UnstructuredTypedObject) SetType(ttype string) {
	u.ObjectType.SetType(ttype)
	if u.Object == nil {
		u.Object = make(map[string]interface{})
	}
	u.Object["type"] = ttype
}

// DeepCopyInto is deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (u *UnstructuredTypedObject) DeepCopyInto(out *UnstructuredTypedObject) {
	*out = *u
	raw := make([]byte, len(u.Raw))
	copy(raw, u.Raw)
	_ = out.setRaw(raw)
}

// DeepCopy is deepcopy function, copying the receiver, creating a new UnstructuredTypedObject.
func (u *UnstructuredTypedObject) DeepCopy() *UnstructuredTypedObject {
	if u == nil {
		return nil
	}
	out := new(UnstructuredTypedObject)
	u.DeepCopyInto(out)
	return out
}

// DecodeInto decodes a unstructured typed object into a TypedObjectAccessor using the default codec
func (u *UnstructuredTypedObject) DecodeInto(into TypedObjectAccessor) error {
	return FromUnstructuredObject(NewDefaultCodec(), u, into)
}

func (u UnstructuredTypedObject) GetRaw() ([]byte, error) {
	data, err := json.Marshal(u.Object)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data, u.Raw) {
		u.Raw = data
	}
	return u.Raw, nil
}

func (u *UnstructuredTypedObject) setRaw(data []byte) error {
	obj := map[string]interface{}{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	u.Raw = data
	u.Object = obj
	return nil
}

// UnmarshalJSON implements a custom json unmarshal method for a unstructured typed object.
func (u *UnstructuredTypedObject) UnmarshalJSON(data []byte) error {
	typedObj := ObjectType{}
	if err := json.Unmarshal(data, &typedObj); err != nil {
		return err
	}

	obj := UnstructuredTypedObject{
		ObjectType: typedObj,
	}
	if err := obj.setRaw(data); err != nil {
		return err
	}
	*u = obj
	return nil
}

// MarshalJSON implements a custom json unmarshal method for a unstructured type.
func (u *UnstructuredTypedObject) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(u.Object)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (_ UnstructuredTypedObject) OpenAPISchemaType() []string { return []string{"object"} }
func (_ UnstructuredTypedObject) OpenAPISchemaFormat() string { return "" }
