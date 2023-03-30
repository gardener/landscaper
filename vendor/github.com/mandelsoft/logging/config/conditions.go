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

package config

import (
	"encoding/json"
	"fmt"

	"github.com/mandelsoft/logging"
)

func init() {
	RegisterCondition("and", And{})
	RegisterCondition("or", Or{})
	RegisterCondition("not", Not{})
	RegisterCondition("tag", Tag(""))
	RegisterCondition("realm", Realm(""))
	RegisterCondition("realmprefix", RealmPrefix(""))
	RegisterCondition("attribute", &Attribute{})
}

////////////////////////////////////////////////////////////////////////////////

type And []json.RawMessage

func (e And) Create(r Registry) (logging.Condition, error) {
	conditions, err := ParseConditions(r, e)
	if err != nil {
		return nil, err
	}
	return logging.And(conditions...), nil
}

////////////////////////////////////////////////////////////////////////////////

type Or And

func (e Or) Create(r Registry) (logging.Condition, error) {
	conditions, err := ParseConditions(r, e)
	if err != nil {
		return nil, err
	}
	return logging.Or(conditions...), nil
}

////////////////////////////////////////////////////////////////////////////////

type Not struct {
	json.RawMessage `json:",inline"`
}

func (e Not) Create(r Registry) (logging.Condition, error) {
	c, err := r.CreateCondition(e.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("cannot parse condition: %w", err)
	}
	return logging.Not(c), nil
}

////////////////////////////////////////////////////////////////////////////////

type Tag string

func (e Tag) Create(_ Registry) (logging.Condition, error) {
	if e == "" {
		return nil, fmt.Errorf("tag name missing")
	}
	return logging.NewTag(string(e)), nil
}

////////////////////////////////////////////////////////////////////////////////

type Realm string

func (e Realm) Create(_ Registry) (logging.Condition, error) {
	if e == "" {
		return nil, fmt.Errorf("realm name missing")
	}
	return logging.NewRealm(string(e)), nil
}

////////////////////////////////////////////////////////////////////////////////

type RealmPrefix string

func (e RealmPrefix) Create(_ Registry) (logging.Condition, error) {
	if e == "" {
		return nil, fmt.Errorf("realm name missing")
	}
	return logging.NewRealmPrefix(string(e)), nil
}

////////////////////////////////////////////////////////////////////////////////

type Attribute struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value,omitempty"`
}

func (e *Attribute) Create(r Registry) (logging.Condition, error) {
	if e.Name == "" {
		return nil, fmt.Errorf("attribute name missing")
	}
	v, err := r.CreateValue(e.Value)
	if err != nil {
		return nil, fmt.Errorf("cannot parse value: %s", err)
	}
	return logging.NewAttribute(e.Name, v), nil
}

////////////////////////////////////////////////////////////////////////////////
