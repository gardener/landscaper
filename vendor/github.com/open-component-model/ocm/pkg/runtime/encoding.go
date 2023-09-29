// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

// github.com/ghodss/yaml

import (
	"encoding/json"

	"sigs.k8s.io/yaml"
)

type Marshaler interface {
	Marshal(obj interface{}) ([]byte, error)
}

type Unmarshaler interface {
	Unmarshal(data []byte, obj interface{}) error
}

type MarshalFunction func(obj interface{}) ([]byte, error)

func (f MarshalFunction) Marshal(obj interface{}) ([]byte, error) { return f(obj) }

type UnmarshalFunction func(data []byte, obj interface{}) error

func (f UnmarshalFunction) Unmarshal(data []byte, obj interface{}) error { return f(data, obj) }

type Encoding interface {
	Unmarshaler
	Marshaler
}

type EncodingWrapper struct {
	Unmarshaler
	Marshaler
}

var DefaultJSONEncoding = &EncodingWrapper{
	Marshaler:   MarshalFunction(json.Marshal),
	Unmarshaler: UnmarshalFunction(json.Unmarshal),
}

var DefaultYAMLEncoding = &EncodingWrapper{
	Marshaler:   MarshalFunction(yaml.Marshal),
	Unmarshaler: UnmarshalFunction(func(data []byte, obj interface{}) error { return yaml.Unmarshal(data, obj) }),
}
