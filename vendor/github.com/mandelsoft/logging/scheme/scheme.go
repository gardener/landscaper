/*
 * Copyright 2022 Mandelsoft. All rights reserved.
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
	"sync"

	"sigs.k8s.io/yaml"
)

type Factory[T any, R any] interface {
	Create(R) (T, error)
}

type Scheme[T any, F any] struct {
	lock           sync.RWMutex
	factoryContext F
	none           T
	prototypes     map[string]reflect.Type
}

func NewScheme[T any, F any](f F) *Scheme[T, F] {
	return &Scheme[T, F]{
		factoryContext: f,
		prototypes:     map[string]reflect.Type{},
	}
}

func (s *Scheme[T, F]) Copy(f F) *Scheme[T, F] {
	s.lock.RLock()
	defer s.lock.RUnlock()

	c := NewScheme[T, F](f)
	for n, p := range s.prototypes {
		c.prototypes[n] = p
	}
	return c
}

func (s *Scheme[T, F]) Register(name string, proto Factory[T, F]) {
	s.lock.Lock()
	defer s.lock.Unlock()

	t := reflect.TypeOf(proto)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	s.prototypes[name] = t
}

func (s *Scheme[T, F]) Get(data []byte) (T, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var m map[string]interface{}

	err := yaml.Unmarshal(data, &m)
	if err != nil {
		return s.none, fmt.Errorf("cannot unmarshal yaml: %w", err)
	}
	if len(m) == 0 {
		return s.none, fmt.Errorf("element missing")
	}
	if len(m) > 1 {
		return s.none, fmt.Errorf("only one key allowed for element")
	}

	var ty string
	var value []byte
	for k, v := range m {
		ty = k
		value, err = json.Marshal(v)
		if err != nil {
			return s.none, err
		}
	}

	t := s.prototypes[ty]
	if t == nil {
		return s.none, fmt.Errorf("unknown element type %q", ty)
	}
	e := reflect.New(t).Interface().(Factory[T, F])
	err = yaml.Unmarshal(value, e)
	if err != nil {
		return s.none, fmt.Errorf("cannot unmarshal type %q: %w", ty, err)
	}
	return e.Create(s.factoryContext)
}
