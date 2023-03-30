// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

// TypeGetter is the interface to be implemented for extracting a type.
type TypeGetter interface {
	// GetType returns the type of the access object.
	GetType() string
}

// TypeSetter is the interface to be implemented for extracting a type.
type TypeSetter interface {
	// SetType sets the type of an abstract element
	SetType(typ string)
}

// TypedObject defines the accessor for a typed object with additional data.
type TypedObject interface {
	TypeGetter
}

var typeTypedObject = reflect.TypeOf((*TypedObject)(nil)).Elem()

// TypedObjectDecoder is able to provide an effective typed object for some
// serilaized form. The technical deserialization is done by an Unmarshaler.
type TypedObjectDecoder interface {
	Decode(data []byte, unmarshaler Unmarshaler) (TypedObject, error)
}

// TypedObjectEncoder is able to provide a versioned representation of
// an effective TypedObject.
type TypedObjectEncoder interface {
	Encode(TypedObject, Marshaler) ([]byte, error)
}

type DirectDecoder struct {
	proto reflect.Type
}

var _ TypedObjectDecoder = &DirectDecoder{}

func MustNewDirectDecoder(proto interface{}) *DirectDecoder {
	d, err := NewDirectDecoder(proto)
	if err != nil {
		panic(err)
	}
	return d
}

func NewDirectDecoder(proto interface{}) (*DirectDecoder, error) {
	t := MustProtoType(proto)
	if !reflect.PtrTo(t).Implements(typeTypedObject) {
		return nil, errors.Newf("object interface %T: must implement TypedObject", proto)
	}
	if t.Kind() != reflect.Struct {
		return nil, errors.Newf("prototype %q must be a struct", t)
	}
	return &DirectDecoder{
		proto: t,
	}, nil
}

func (d *DirectDecoder) CreateInstance() TypedObject {
	return reflect.New(d.proto).Interface().(TypedObject)
}

func (d *DirectDecoder) Decode(data []byte, unmarshaler Unmarshaler) (TypedObject, error) {
	inst := d.CreateInstance()
	err := unmarshaler.Unmarshal(data, inst)
	if err != nil {
		return nil, err
	}

	return inst, nil
}

func (d *DirectDecoder) Encode(obj TypedObject, marshaler Marshaler) ([]byte, error) {
	return marshaler.Marshal(obj)
}

// TypedObjectConverter converts a versioned representation into the
// intended type required by the scheme.
type TypedObjectConverter interface {
	ConvertTo(in interface{}) (TypedObject, error)
}

// ConvertingDecoder uses a serialization from different from the
// intended object type, that is converted to achieve the decode result.
type ConvertingDecoder struct {
	proto reflect.Type
	TypedObjectConverter
}

var _ TypedObjectDecoder = &ConvertingDecoder{}

func MustNewConvertingDecoder(proto interface{}, conv TypedObjectConverter) *ConvertingDecoder {
	d, err := NewConvertingDecoder(proto, conv)
	if err != nil {
		panic(err)
	}
	return d
}

func NewConvertingDecoder(proto interface{}, conv TypedObjectConverter) (*ConvertingDecoder, error) {
	t, err := ProtoType(proto)
	if err != nil {
		return nil, err
	}
	return &ConvertingDecoder{
		proto:                t,
		TypedObjectConverter: conv,
	}, nil
}

func (d *ConvertingDecoder) Decode(data []byte, unmarshaler Unmarshaler) (TypedObject, error) {
	versioned := d.CreateData()
	err := unmarshaler.Unmarshal(data, versioned)
	if err != nil {
		return nil, err
	}
	return d.ConvertTo(versioned)
}

func (d *ConvertingDecoder) CreateData() interface{} {
	return reflect.New(d.proto).Interface()
}

// KnownTypes is a set of known type names mapped to appropriate object decoders.
type KnownTypes map[string]TypedObjectDecoder

// Copy provides a copy of the actually known types.
func (t KnownTypes) Copy() KnownTypes {
	n := KnownTypes{}
	for k, v := range t {
		n[k] = v
	}
	return n
}

// TypeNames return a sorted list of known type names.
func (t KnownTypes) TypeNames() []string {
	types := make([]string, 0, len(t))
	for t := range t {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// Scheme is the interface to describe a set of object types
// that implement a dedicated interface.
// As such it knows about the desired interface of the instances
// and can validate it. Additionally, it provides an implementation
// for generic unstructured objects that can be used to decode
// any serialized from of object candidates and provide the
// effective type.
type Scheme interface {
	RegisterByDecoder(typ string, decoder TypedObjectDecoder) error

	ValidateInterface(object TypedObject) error
	CreateUnstructured() Unstructured
	Convert(object TypedObject) (TypedObject, error)
	GetDecoder(otype string) TypedObjectDecoder
	Decode(data []byte, unmarshaler Unmarshaler) (TypedObject, error)
	Encode(obj TypedObject, marshaler Marshaler) ([]byte, error)
	EnforceDecode(data []byte, unmarshaler Unmarshaler) (TypedObject, error)
	KnownTypes() KnownTypes
	KnownTypeNames() []string
}

type SchemeBase interface {
	AddKnownTypes(scheme Scheme)
	Scheme
}
type defaultScheme struct {
	lock           sync.RWMutex
	base           Scheme
	instance       reflect.Type
	unstructured   reflect.Type
	defaultdecoder TypedObjectDecoder
	acceptUnknown  bool
	types          KnownTypes
}

type BaseScheme interface {
	BaseScheme() Scheme
}

var _ BaseScheme = (*defaultScheme)(nil)

func MustNewDefaultScheme(protoIfce interface{}, protoUnstr Unstructured, acceptUnknown bool, defaultdecoder TypedObjectDecoder, base ...Scheme) SchemeBase {
	return utils.Must(NewDefaultScheme(protoIfce, protoUnstr, acceptUnknown, defaultdecoder, base...))
}

func NewDefaultScheme(protoIfce interface{}, protoUnstr Unstructured, acceptUnknown bool, defaultdecoder TypedObjectDecoder, base ...Scheme) (SchemeBase, error) {
	if protoIfce == nil {
		return nil, fmt.Errorf("object interface must be given by pointer to interacted (is nil)")
	}
	it := reflect.TypeOf(protoIfce)
	if it.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("object interface %T: must be given by pointer to interacted (is not pointer)", protoIfce)
	}
	it = it.Elem()
	if it.Kind() != reflect.Interface {
		return nil, fmt.Errorf("object interface %T: must be given by pointer to interacted (does not point to interface)", protoIfce)
	}
	if !it.Implements(typeTypedObject) {
		return nil, fmt.Errorf("object interface %T: must implement TypedObject", protoIfce)
	}

	ut, err := ProtoType(protoUnstr)
	if err != nil {
		return nil, errors.Wrapf(err, "unstructured prototype %T", protoUnstr)
	}
	if acceptUnknown {
		if !reflect.PtrTo(ut).Implements(typeTypedObject) {
			return nil, fmt.Errorf("unstructured type %T must implement TypedObject to be acceptale as unknown result", protoUnstr)
		}
	}

	return &defaultScheme{
		base:           utils.Optional(base...),
		instance:       it,
		unstructured:   ut,
		defaultdecoder: defaultdecoder,
		types:          KnownTypes{},
		acceptUnknown:  acceptUnknown,
	}, nil
}

func (d *defaultScheme) BaseScheme() Scheme {
	return d.base
}

func (d *defaultScheme) AddKnownTypes(s Scheme) {
	d.lock.Lock()
	defer d.lock.Unlock()
	for k, v := range s.KnownTypes() {
		d.types[k] = v
	}
}

func (d *defaultScheme) KnownTypes() KnownTypes {
	d.lock.RLock()
	defer d.lock.RUnlock()
	if d.base == nil {
		return d.types.Copy()
	}
	kt := d.base.KnownTypes()
	for n, t := range d.types {
		kt[n] = t
	}
	return kt
}

// KnownTypeNames return a sorted list of known type names.
func (d *defaultScheme) KnownTypeNames() []string {
	d.lock.RLock()
	defer d.lock.RUnlock()

	types := make([]string, 0, len(d.types))
	for t := range d.types {
		types = append(types, t)
	}
	if d.base != nil {
		types = append(types, d.base.KnownTypeNames()...)
	}
	sort.Strings(types)
	return types
}

func RegisterByType(s Scheme, typ string, proto TypedObject) error {
	t, err := NewDirectDecoder(proto)
	if err != nil {
		return err
	}
	return s.RegisterByDecoder(typ, t)
}

func (d *defaultScheme) RegisterByDecoder(typ string, decoder TypedObjectDecoder) error {
	if decoder == nil {
		return errors.Newf("decoder must be given")
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	d.types[typ] = decoder
	return nil
}

func (d *defaultScheme) ValidateInterface(object TypedObject) error {
	t := reflect.TypeOf(object)
	if !t.Implements(d.instance) {
		return errors.Newf("object type %q does not implement required instance interface %q", t, d.instance)
	}
	return nil
}

func (d *defaultScheme) GetDecoder(typ string) TypedObjectDecoder {
	d.lock.RLock()
	defer d.lock.RUnlock()
	decoder := d.types[typ]
	if decoder == nil && d.base != nil {
		decoder = d.base.GetDecoder(typ)
	}
	return decoder
}

func (d *defaultScheme) CreateUnstructured() Unstructured {
	return reflect.New(d.unstructured).Interface().(Unstructured)
}

func (d *defaultScheme) Encode(obj TypedObject, marshaler Marshaler) ([]byte, error) {
	if marshaler == nil {
		marshaler = DefaultYAMLEncoding
	}
	decoder := d.GetDecoder(obj.GetType())
	if encoder, ok := decoder.(TypedObjectEncoder); ok {
		return encoder.Encode(obj, marshaler)
	}
	return marshaler.Marshal(obj)
}

func (d *defaultScheme) Decode(data []byte, unmarshal Unmarshaler) (TypedObject, error) {
	un := d.CreateUnstructured()
	if unmarshal == nil {
		unmarshal = DefaultYAMLEncoding
	}
	err := unmarshal.Unmarshal(data, un)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal unstructured")
	}
	if un.GetType() == "" {
		/*
			if d.acceptUnknown {
				return un.(TypedObject), nil
			}
		*/
		return nil, errors.Newf("no type found")
	}
	decoder := d.GetDecoder(un.GetType())
	if decoder == nil {
		if d.defaultdecoder != nil {
			o, err := d.defaultdecoder.Decode(data, unmarshal)
			if err == nil {
				if o != nil {
					return o, nil
				}
			} else if !errors.IsErrUnknownKind(err, errors.KIND_OBJECTTYPE) {
				return nil, err
			}
		}
		if d.acceptUnknown {
			return un.(TypedObject), nil
		}
		return nil, errors.ErrUnknown(errors.KIND_OBJECTTYPE, un.GetType())
	}
	return decoder.Decode(data, unmarshal)
}

func (d *defaultScheme) EnforceDecode(data []byte, unmarshal Unmarshaler) (TypedObject, error) {
	un := d.CreateUnstructured()
	if unmarshal == nil {
		unmarshal = DefaultYAMLEncoding.Unmarshaler
	}
	err := unmarshal.Unmarshal(data, un)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal unstructured")
	}
	if un.GetType() == "" {
		if d.acceptUnknown {
			return un.(TypedObject), nil
		}
		return un.(TypedObject), errors.Newf("no type found")
	}
	decoder := d.GetDecoder(un.GetType())
	if decoder == nil {
		if d.defaultdecoder != nil {
			o, err := d.defaultdecoder.Decode(data, unmarshal)
			if err == nil {
				return o, nil
			}
			if !errors.IsErrUnknownKind(err, errors.KIND_OBJECTTYPE) {
				return un.(TypedObject), err
			}
		}
		if d.acceptUnknown {
			return un.(TypedObject), nil
		}
		return un.(TypedObject), errors.ErrUnknown(errors.KIND_OBJECTTYPE, un.GetType())
	}
	o, err := decoder.Decode(data, unmarshal)
	if err != nil {
		return un.(TypedObject), err
	}
	return o, err
}

func (d *defaultScheme) Convert(o TypedObject) (TypedObject, error) {
	if o.GetType() == "" {
		return nil, errors.Newf("no type found")
	}
	data, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	decoder := d.GetDecoder(o.GetType())
	if decoder == nil {
		if d.defaultdecoder != nil {
			object, err := d.defaultdecoder.Decode(data, DefaultJSONEncoding)
			if err == nil {
				return object, nil
			}
			if !errors.IsErrUnknownKind(err, errors.KIND_OBJECTTYPE) {
				return nil, err
			}
		}
		return nil, errors.ErrUnknown(errors.KIND_OBJECTTYPE, o.GetType())
	}
	r, err := decoder.Decode(data, DefaultJSONEncoding)
	if err != nil {
		return nil, err
	}
	if reflect.TypeOf(r) == reflect.TypeOf(o) {
		return o, nil
	}
	return r, nil
}
