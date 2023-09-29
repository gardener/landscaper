/*
 * Copyright 2023 Mandelsoft. All rights reserved.
 *  This file is licensed under the Apache Software License, v. 2 except as noted
 *  otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package scheme

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Element[T any] struct {
	typ  string
	spec T
	raw  []byte
}

var _ json.Marshaler = (*Element[any])(nil)

var _ json.Unmarshaler = (*Element[any])(nil)

func NewElement[T any](typ string, spec T) Element[T] {
	return Element[T]{
		typ:  typ,
		spec: spec,
	}
}

func (e Element[T]) MarshalJSON() ([]byte, error) {
	if reflect.ValueOf(e.spec).IsValid() && !reflect.ValueOf(e.spec).IsZero() {
		data, err := json.Marshal(e.spec)
		if err != nil {
			return nil, err
		}
		e.raw = data
	}
	v := map[string]json.RawMessage{
		e.typ: e.raw,
	}
	return json.Marshal(v)
}

func (e *Element[T]) UnmarshalJSON(bytes []byte) error {
	var zero T
	var v map[string]json.RawMessage

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}
	if len(v) == 0 {
		return fmt.Errorf("element type missing")
	}
	e.spec = zero
	e.raw = nil
	for k, c := range v {
		if e.raw != nil {
			return fmt.Errorf("logging config element may have only name entry")
		}
		e.raw = c
		e.typ = k
	}
	return nil
}
