// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/modern-go/reflect2"
	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/errors"
)

const ATTR_TYPE = "type"

// ATTENTION: UnstructuredTypedObject CANNOT be used as anonymous
// field together with the default struct marshalling with the
// great json marshallers.
// Anonymous inline struct fields are always marshaled by the default struct
// marshales in a depth first manner without observing the Marshal interface!!!!
//
// Therefore all structs in this module deriving from UnstructuedTypedObject
// are explicitly implementing the marshal/unmarshal interface.
//
// Side Fact: Marshaling a map[interface{}] filled by unmarshaling a marshaled
// object with anonymous fields is not stable, because the inline fields
// are sorted depth firt for marshalling, while maps key are marshaled
// completely in order.
// Therefore we do not store the raw bytes but marshal them always from
// the UnstructuedMap.

// Unstructured is the interface to represent generic object data for
// types handled by schemes.
type Unstructured interface {
	TypeGetter
	GetRaw() ([]byte, error)
}

type Object interface{}

type JSONMarhaler interface {
	MarshalJSON() ([]byte, error)
}

// UnstructuredMap is a generic data map.
type UnstructuredMap map[string]interface{}

// FlatMerge just joins the direct attribute set.
func (m UnstructuredMap) FlatMerge(o UnstructuredMap) UnstructuredMap {
	for k, v := range o {
		m[k] = v
	}
	return m
}

// FlatCopy just copies the attributes.
func (m UnstructuredMap) FlatCopy() UnstructuredMap {
	r := UnstructuredMap{}
	for k, v := range m {
		r[k] = v
	}
	return r
}

func (m UnstructuredMap) Match(o UnstructuredMap) bool {
	for k, v := range m {
		vo := o[k]
		if !matchValue(v, vo) {
			return false
		}
	}
	for k, v := range o {
		if _, ok := m[k]; ok {
			continue
		}
		if !matchValue(v, nil) {
			return false
		}
	}
	return true
}

func matchValue(a, b interface{}) bool {
	if a == nil {
		if b == nil {
			return true
		}
		a, b = b, a
	}
	// a in not nil
	if b != nil {
		if reflect.TypeOf(a) != reflect.TypeOf(b) {
			return false
		}
		switch v := a.(type) {
		case []interface{}:
			if len(v) != len(b.([]interface{})) {
				return false
			}
			for i, e := range b.([]interface{}) {
				if !matchValue(v[i], e) {
					return false
				}
			}
			return true
		case map[string]interface{}:
			return UnstructuredMap(v).Match(UnstructuredMap(b.(map[string]interface{})))
		default:
			return reflect.DeepEqual(a, b)
		}
	}
	// check initial (b==nil)
	switch v := a.(type) {
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return reflect.ValueOf(a).IsZero()
	}
}

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
func TypedObjectEqual(a, b TypedObject) bool {
	if a.GetType() != b.GetType() {
		return false
	}
	uA, err := ToUnstructuredTypedObject(a)
	if err != nil {
		return false
	}
	uB, err := ToUnstructuredTypedObject(b)
	if err != nil {
		return false
	}
	return UnstructuredTypesEqual(uA, uB)
}

// NewEmptyUnstructured creates a new typed object without additional data.
func NewEmptyUnstructured(ttype string) *UnstructuredTypedObject {
	return NewUnstructuredType(ttype, nil)
}

// NewEmptyUnstructuredVersioned creates a new typed object without additional data.
func NewEmptyUnstructuredVersioned(ttype string) *UnstructuredVersionedTypedObject {
	return &UnstructuredVersionedTypedObject{*NewUnstructuredType(ttype, nil)}
}

// NewUnstructuredType creates a new unstructured typed object.
func NewUnstructuredType(ttype string, data UnstructuredMap) *UnstructuredTypedObject {
	unstr := &UnstructuredTypedObject{}
	unstr.Object = data
	unstr.SetType(ttype)
	return unstr
}

// UnstructuredConverter converts the actual object to an UnstructuredTypedObject.
type UnstructuredConverter interface {
	ToUnstructured() (*UnstructuredTypedObject, error)
}

// UnstructuredTypedObject describes a generic typed object.
// +kubebuilder:pruning:PreserveUnknownFields
type UnstructuredTypedObject struct {
	ObjectType `json:",inline"`
	Object     UnstructuredMap `json:"-"`
}

func (s *UnstructuredTypedObject) ToUnstructured() (*UnstructuredTypedObject, error) {
	return s, nil
}

func (u *UnstructuredTypedObject) SetType(ttype string) {
	u.ObjectType.SetType(ttype)
	if u.Object == nil {
		u.Object = UnstructuredMap{}
	}
	u.Object[ATTR_TYPE] = ttype
}

// DeepCopyInto is deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (u *UnstructuredTypedObject) DeepCopyInto(out **UnstructuredTypedObject) {
	*out = new(UnstructuredTypedObject)
	**out = *u

	raw, err := json.Marshal(u.Object)
	if err != nil {
		logrus.Error(err)
	}

	_ = (*out).setRaw(raw)
}

// DeepCopy is a deepcopy function, copying the receiver, creating a new UnstructuredTypedObject.
func (u *UnstructuredTypedObject) DeepCopy() *UnstructuredTypedObject {
	if u == nil {
		return nil
	}
	var out *UnstructuredTypedObject
	u.DeepCopyInto(&out)
	return out
}

func (u UnstructuredTypedObject) GetRaw() ([]byte, error) {
	return json.Marshal(u.Object)
}

func (u *UnstructuredTypedObject) setRaw(data []byte) error {
	obj := UnstructuredMap{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	u.Object = obj
	return nil
}

// EvaluateUnstructured converts an unstructured object into a typed object.
// Go does not support generic methods.
func EvaluateUnstructured[T TypedObject, R TypedObjectDecoder[T]](u *UnstructuredTypedObject, types Scheme[T, R]) (T, error) {
	var zero T

	data, err := u.GetRaw()
	if err != nil {
		return zero, fmt.Errorf("unable to get data from unstructured object: %w", err)
	}
	var decoder TypedObjectDecoder[T]
	if types != nil {
		decoder = types.GetDecoder(u.GetType())
	}
	if decoder == nil {
		return zero, errors.ErrUnknown(errors.KIND_OBJECTTYPE, u.GetType())
	}

	if obj, err := decoder.Decode(data, DefaultJSONEncoding); err != nil {
		return zero, fmt.Errorf("unable to decode object %q: %w", u.GetType(), err)
	} else {
		return obj, nil
	}
}

// UnmarshalJSON implements a custom json unmarshal method for a unstructured typed object.
func (u *UnstructuredTypedObject) UnmarshalJSON(data []byte) error {
	logrus.Debugf("unmarshal raw: %s\n", string(data))
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
func (u UnstructuredTypedObject) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(u.Object)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (_ UnstructuredTypedObject) OpenAPISchemaType() []string { return []string{"object"} }
func (_ UnstructuredTypedObject) OpenAPISchemaFormat() string { return "" }

////////////////////////////////////////////////////////////////////////////////
// Utils
////////////////////////////////////////////////////////////////////////////////

// ToUnstructuredTypedObject converts a typed object to a unstructured object.
func ToUnstructuredTypedObject(obj TypedObject) (*UnstructuredTypedObject, error) {
	if reflect2.IsNil(obj) {
		return nil, nil
	}
	if un, ok := obj.(UnstructuredConverter); ok {
		return un.ToUnstructured()
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	uObj := &UnstructuredTypedObject{}
	if err := json.Unmarshal(data, uObj); err != nil {
		return nil, err
	}
	return uObj, nil
}

// ToUnstructuredObject converts any object into a structure map.
func ToUnstructuredObject(obj interface{}) (UnstructuredMap, error) {
	if reflect2.IsNil(obj) {
		return nil, nil
	}
	if un, ok := obj.(map[string]interface{}); ok {
		return UnstructuredMap(un), nil
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	uObj := UnstructuredMap{}
	if err := json.Unmarshal(data, &uObj); err != nil {
		return nil, err
	}
	return uObj, nil
}

type UnstructuredTypedObjectList []*UnstructuredTypedObject

func (l UnstructuredTypedObjectList) Copy() UnstructuredTypedObjectList {
	n := make(UnstructuredTypedObjectList, len(l))
	for i, u := range l {
		copied := *u
		n[i] = &copied
	}
	return n
}
