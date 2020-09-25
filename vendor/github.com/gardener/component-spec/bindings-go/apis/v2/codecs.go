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

package v2

import (
	"fmt"
	"strings"
)

// KnownTypes defines a set of known types.
type KnownTypes map[string]TypedObjectCodec

// KnownTypeValidationFunc defines a function that can optionally validate types.
type KnownTypeValidationFunc func(ttype string) error

// TypedObjectCodec describes a known component type and how it is decoded and encoded
type TypedObjectCodec interface {
	TypedObjectDecoder
	TypedObjectEncoder
}

// TypedObjectCodecWrapper is a simple struct that implements the TypedObjectCodec interface
type TypedObjectCodecWrapper struct {
	TypedObjectDecoder
	TypedObjectEncoder
}

// TypedObjectDecoder defines a decoder for a typed object.
type TypedObjectDecoder interface {
	Decode(data []byte) (TypedObjectAccessor, error)
}

// TypedObjectEncoder defines a encoder for a typed object.
type TypedObjectEncoder interface {
	Encode(accessor TypedObjectAccessor) ([]byte, error)
}

// TypedObjectDecoderFunc is a simple function that implements the TypedObjectDecoder interface.
type TypedObjectDecoderFunc func(data []byte) (TypedObjectAccessor, error)

// Decode is the Decode implementation of the TypedObjectDecoder interface.
func (e TypedObjectDecoderFunc) Decode(data []byte) (TypedObjectAccessor, error) {
	return e(data)
}

// TypedObjectEncoderFunc is a simple function that implements the TypedObjectEncoder interface.
type TypedObjectEncoderFunc func(accessor TypedObjectAccessor) ([]byte, error)

// Encode is the Encode implementation of the TypedObjectEncoder interface.
func (e TypedObjectEncoderFunc) Encode(accessor TypedObjectAccessor) ([]byte, error) {
	return e(accessor)
}

// ValidateAccessType validates that a type is known or of a generic type.
// todo: revisit; currently "x-" specifies a generic type
func ValidateAccessType(ttype string) error {
	if _, ok := KnownAccessTypes[ttype]; ok {
		return nil
	}

	if !strings.HasPrefix(ttype, "x-") {
		return fmt.Errorf("unknown non generic types %s", ttype)
	}
	return nil
}
