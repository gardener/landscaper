// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"github.com/modern-go/reflect2"
)

type UnstructuredVersionedTypedObject struct {
	UnstructuredTypedObject `json:",inline"`
}

func ToUnstructuredVersionedTypedObject(obj TypedObject) (*UnstructuredVersionedTypedObject, error) {
	if reflect2.IsNil(obj) {
		return nil, nil
	}
	if v, ok := obj.(*UnstructuredVersionedTypedObject); ok {
		return v, nil
	}
	u, err := ToUnstructuredTypedObject(obj)
	if err != nil {
		return nil, err
	}
	return &UnstructuredVersionedTypedObject{*u}, nil
}

func (s *UnstructuredVersionedTypedObject) ToUnstructured() (*UnstructuredTypedObject, error) {
	return &s.UnstructuredTypedObject, nil
}

func (s *UnstructuredVersionedTypedObject) GetKind() string {
	return ObjectVersionedType(s.ObjectType).GetKind()
}

func (s *UnstructuredVersionedTypedObject) GetVersion() string {
	return ObjectVersionedType(s.ObjectType).GetVersion()
}

func (u *UnstructuredVersionedTypedObject) DeepCopy() *UnstructuredVersionedTypedObject {
	if u == nil {
		return nil
	}
	return &UnstructuredVersionedTypedObject{
		*u.UnstructuredTypedObject.DeepCopy(),
	}
}

var _ VersionedTypedObject = &UnstructuredVersionedTypedObject{}
