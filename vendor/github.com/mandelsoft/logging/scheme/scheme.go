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
	var zero T
	var e Element[Factory[T, F]]
	err := yaml.Unmarshal(data, &e)
	if err != nil {
		return zero, err
	}
	return s.GetFromElement(&e)
}

func (s *Scheme[T, F]) GetFromElement(d *Element[Factory[T, F]]) (T, error) {
	var zero T

	s.lock.RLock()
	defer s.lock.RUnlock()

	t := s.prototypes[d.typ]
	if t == nil {
		return s.none, fmt.Errorf("unknown element type %q", d.typ)
	}
	f := reflect.New(t).Interface().(Factory[T, F])
	if d.raw != nil {
		err := yaml.Unmarshal(d.raw, f)
		if err != nil {
			return s.none, fmt.Errorf("cannot unmarshal type %q: %w", d.typ, err)
		}
	} else {
		f = d.spec
	}
	e, err := f.Create(s.factoryContext)
	if err != nil {
		return zero, err
	}
	d.spec = f
	d.raw = nil
	return e, nil
}
