// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package scheme

import (
	"reflect"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type Converter[O Object] interface {
	ConvertTo(Object) (O, error)
	ConvertFrom(O) (Object, error)
}

type runtimeConverterAdapter[O Object] struct {
	converter Converter[O]
}

func (c *runtimeConverterAdapter[O]) ConvertTo(object interface{}) (runtime.TypedObject, error) {
	return c.converter.ConvertTo(object.(Object))
}

type defaultType[O Object] struct {
	decoder   *runtime.ConvertingDecoder
	converter Converter[O]
}

func NewTypeByProtoType[O Object](proto Object, converter Converter[O]) Type[O] {
	return &defaultType[O]{
		decoder:   runtime.MustNewConvertingDecoder(proto, &runtimeConverterAdapter[O]{converter}),
		converter: converter,
	}
}

func (t *defaultType[O]) Decode(data []byte, unmarshaler runtime.Unmarshaler) (O, error) {
	var zero O
	o, err := t.decoder.Decode(data, unmarshaler)
	if err != nil {
		return zero, err
	}
	return o.(O), nil
}

func (t *defaultType[O]) Encode(o O, m runtime.Marshaler) ([]byte, error) {
	c, err := t.converter.ConvertFrom(o)
	if err != nil {
		return nil, err
	}
	return m.Marshal(c)
}

////////////////////////////////////////////////////////////////////////////////

type IdentityConverter[O Object] struct{}

func (c IdentityConverter[O]) ConvertFrom(o O) (Object, error) {
	return o, nil
}

func (c IdentityConverter[O]) ConvertTo(o Object) (O, error) {
	var zero O
	if s, ok := o.(O); ok {
		return s, nil
	}
	return zero, errors.ErrInvalid("raw type", reflect.TypeOf(o).String())
}

func NewIdentityType[O Object](proto O) Type[O] {
	return NewTypeByProtoType[O](proto, IdentityConverter[O]{})
}
