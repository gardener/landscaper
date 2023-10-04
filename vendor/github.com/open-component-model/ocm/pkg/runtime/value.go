// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"encoding/json"
	"reflect"

	"github.com/open-component-model/ocm/pkg/errors"
)

// RawValue is a json-encoded representation of a value object like
// a json.RawMessage, but additionally offers Getter and Setter.
type RawValue struct {
	json.RawMessage `json:",inline"`
}

func (in RawValue) Copy() RawValue {
	dst := make([]byte, len(in.RawMessage))
	copy(dst, in.RawMessage)
	return RawValue{dst}
}

// GetValue returns the value as parsed object.
func (in *RawValue) GetValue(dest interface{}) error {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &json.InvalidUnmarshalError{reflect.TypeOf(dest)}
	}
	// set initial value fist to avoid merging of content
	rv.Elem().Set(reflect.Zero(rv.Elem().Type()))
	return json.Unmarshal(in.RawMessage, dest)
}

// SetValue sets the value by marshalling the given object.
// A passed byte slice is validated to be valid json.
func (in *RawValue) SetValue(value interface{}) error {
	raw, err := AsRawMessage(value)
	if err != nil {
		return err
	}
	in.RawMessage = raw
	return nil
}

func AsRawMessage(value interface{}) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	var (
		data []byte
		ok   bool
		err  error
	)

	if data, ok = value.([]byte); ok {
		var v interface{}
		err = json.Unmarshal(data, &v)
		if err != nil {
			return nil, errors.ErrInvalid("value", string(data))
		}
	} else {
		data, err = json.Marshal(value)
		if err != nil {
			return nil, errors.ErrInvalid("value", "<object>")
		}
	}
	return data, nil
}
