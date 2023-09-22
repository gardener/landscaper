// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package datacontext

import (
	"sort"
	"sync"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type AttributeType interface {
	Name() string
	Decode(data []byte, unmarshaler runtime.Unmarshaler) (interface{}, error)
	Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error)
	Description() string
}

// Converter is an optional interface an AttributeType can implement to
// normalize an attribute value. It is called by the Attributes.SetAttribute
// method.
type Converter interface {
	Convert(interface{}) (interface{}, error)
}

type AttributeScheme interface {
	Register(name string, typ AttributeType, short ...string) error

	Decode(attr string, data []byte, unmarshaler runtime.Unmarshaler) (interface{}, error)
	Encode(attr string, v interface{}, marshaller runtime.Marshaler) ([]byte, error)
	Convert(attr string, v interface{}) (interface{}, error)
	GetType(attr string) (AttributeType, error)

	AddKnownTypes(scheme AttributeScheme)
	Shortcuts() common.Properties
	KnownTypes() KnownTypes
	KnownTypeNames() []string
}

var DefaultAttributeScheme = NewDefaulAttritutetScheme()

// KnownTypes is a set of known type names mapped to appropriate object decoders.
type KnownTypes map[string]AttributeType

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

type defaultScheme struct {
	lock  sync.RWMutex
	types KnownTypes
	short common.Properties
}

func NewDefaulAttritutetScheme() AttributeScheme {
	return &defaultScheme{
		types: KnownTypes{},
		short: common.Properties{},
	}
}

func (d *defaultScheme) AddKnownTypes(s AttributeScheme) {
	d.lock.Lock()
	defer d.lock.Unlock()
	for k, v := range s.KnownTypes() {
		d.types[k] = v
	}
	for k, v := range s.Shortcuts() {
		d.short[k] = v
	}
}

func (d *defaultScheme) KnownTypes() KnownTypes {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.types.Copy()
}

func (d *defaultScheme) Shortcuts() common.Properties {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.short.Copy()
}

// KnownTypeNames return a sorted list of known type names.
func (d *defaultScheme) KnownTypeNames() []string {
	d.lock.RLock()
	defer d.lock.RUnlock()
	types := make([]string, 0, len(d.types))
	for t := range d.types {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func RegisterAttributeType(name string, typ AttributeType, short ...string) error {
	return DefaultAttributeScheme.Register(name, typ, short...)
}

func (d *defaultScheme) Register(name string, typ AttributeType, short ...string) error {
	if typ == nil {
		return errors.Newf("type object must be given")
	}
	if name == "" {
		return errors.Newf("name must be given")
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	d.types[name] = typ
	for _, s := range short {
		d.short[s] = name
	}
	return nil
}

func (d *defaultScheme) getType(attr string) AttributeType {
	if s, ok := d.short[attr]; ok {
		attr = s
	}
	return d.types[attr]
}

func (d *defaultScheme) GetType(attr string) (AttributeType, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	t := d.getType(attr)
	if t == nil {
		return nil, errors.ErrUnknown("attribute", attr)
	}
	return t, nil
}

func (d *defaultScheme) Convert(attr string, value interface{}) (interface{}, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	t := d.getType(attr)
	if t == nil {
		return nil, errors.ErrUnknown("attribute", attr)
	}
	if c, ok := t.(Converter); ok {
		return c.Convert(value)
	}
	return value, nil
}

func (d *defaultScheme) Encode(attr string, value interface{}, marshaler runtime.Marshaler) ([]byte, error) {
	if marshaler == nil {
		marshaler = runtime.DefaultJSONEncoding
	}
	d.lock.RLock()
	defer d.lock.RUnlock()
	t := d.getType(attr)
	if t == nil {
		return nil, errors.ErrUnknown("attribute", attr)
	}
	return t.Encode(value, marshaler)
}

func (d *defaultScheme) Decode(attr string, data []byte, unmarshaler runtime.Unmarshaler) (interface{}, error) {
	if unmarshaler == nil {
		unmarshaler = runtime.DefaultJSONEncoding
	}
	d.lock.RLock()
	defer d.lock.RUnlock()
	t := d.getType(attr)
	if t == nil {
		return nil, errors.ErrUnknown("attribute", attr)
	}
	return t.Decode(data, unmarshaler)
}

type DefaultAttributeType struct{}

func (_ DefaultAttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	return marshaller.Marshal(v)
}
