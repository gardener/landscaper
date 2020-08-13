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

// KnownAccessTypes contains all known access serializer
var KnownAccessTypes = map[string]AccessCodec{
	OCIRegistryType: ociCodec,
	WebType:         webCodec,
}

// AccessCodec describes a known component type and how it is decoded and encoded
type AccessCodec interface {
	AccessDecoder
	AccessEncoder
}

// AccessCodecWrapper is a simple struct that implements the AccessCodec interface
type AccessCodecWrapper struct {
	AccessDecoder
	AccessEncoder
}

// AccessDecoder decodes a component dependency.
type AccessDecoder interface {
	Decode(data []byte) (AccessAccessor, error)
}

// AccessEncoder encodes a component dependency.
type AccessEncoder interface {
	Encode(accessor AccessAccessor) ([]byte, error)
}

// AccessDecoderFunc is a simple function that implements the AccessDecoder interface.
type AccessDecoderFunc func(data []byte) (AccessAccessor, error)

// Decode is the Decode implementation of the AccessDecoder interface.
func (e AccessDecoderFunc) Decode(data []byte) (AccessAccessor, error) {
	return e(data)
}

// AccessEncoderFunc is a simple function that implements the AccessEncoder interface.
type AccessEncoderFunc func(accessor AccessAccessor) ([]byte, error)

// Encode is the Encode implementation of the AccessEncoder interface.
func (e AccessEncoderFunc) Encode(accessor AccessAccessor) ([]byte, error) {
	return e(accessor)
}

// ValidateAccessType validates that a type is known or of a generic type.
// todo: revisit currently "x-" specifies a generic type
func ValidateAccessType(ttype string) error {
	if _, ok := KnownAccessTypes[ttype]; ok {
		return nil
	}

	if !strings.HasPrefix(ttype, "x-") {
		return fmt.Errorf("unknown non generic types %s", ttype)
	}
	return nil
}
