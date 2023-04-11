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
	"fmt"

	"github.com/mandelsoft/logging"
	"github.com/mandelsoft/logging/scheme"
)

func init() {
	RegisterCondition("and", AndType{})
	RegisterCondition("or", OrType{})
	RegisterCondition("not", &NotType{})
	RegisterCondition("tag", TagType(""))
	RegisterCondition("realm", RealmType(""))
	RegisterCondition("realmprefix", RealmPrefixType(""))
	RegisterCondition("attribute", &AttributeType{})
}

func newCondition(typ string, v ConditionType) Condition {
	return scheme.NewElement(typ, v)
}

////////////////////////////////////////////////////////////////////////////////

// AndType should be []Condition, but this does not work with Go Generics
// so, just explode the type, and it magically works.
type AndType []scheme.Element[scheme.Factory[logging.Condition, Registry]]

func And(conds ...Condition) Condition {
	s := AndType(conds)
	return newCondition("and", &s)
}

func (e AndType) Create(r Registry) (logging.Condition, error) {
	conditions, err := ParseConditions(r, e)
	if err != nil {
		return nil, err
	}
	return logging.And(conditions...), nil
}

////////////////////////////////////////////////////////////////////////////////

type OrType AndType

func Or(conds ...Condition) Condition {
	s := OrType(conds)
	return newCondition("or", &s)
}

func (e OrType) Create(r Registry) (logging.Condition, error) {
	conditions, err := ParseConditions(r, e)
	if err != nil {
		return nil, err
	}
	return logging.Or(conditions...), nil
}

////////////////////////////////////////////////////////////////////////////////

type NotType struct {
	Condition `json:",inline"`
}

func Not(c Condition) Condition {
	s := NotType{c}
	return newCondition("not", &s)
}

func (e *NotType) Create(r Registry) (logging.Condition, error) {
	c, err := r.CreateConditionFromElement(&e.Condition)
	if err != nil {
		return nil, fmt.Errorf("cannot parse condition: %w", err)
	}
	return logging.Not(c), nil
}

////////////////////////////////////////////////////////////////////////////////

type TagType string

func Tag(tag string) Condition {
	s := TagType(tag)
	return newCondition("tag", &s)
}

func (e TagType) Create(_ Registry) (logging.Condition, error) {
	if e == "" {
		return nil, fmt.Errorf("tag name missing")
	}
	return logging.NewTag(string(e)), nil
}

////////////////////////////////////////////////////////////////////////////////

type RealmType string

func Realm(tag string) Condition {
	s := RealmType(tag)
	return newCondition("realm", &s)
}

func (e RealmType) Create(_ Registry) (logging.Condition, error) {
	if e == "" {
		return nil, fmt.Errorf("realm name missing")
	}
	return logging.NewRealm(string(e)), nil
}

////////////////////////////////////////////////////////////////////////////////

type RealmPrefixType string

func RealmPrefix(tag string) Condition {
	s := RealmPrefixType(tag)
	return newCondition("realmprefix", &s)
}

func (e RealmPrefixType) Create(_ Registry) (logging.Condition, error) {
	if e == "" {
		return nil, fmt.Errorf("realm name missing")
	}
	return logging.NewRealmPrefix(string(e)), nil
}

////////////////////////////////////////////////////////////////////////////////

type AttributeType struct {
	Name  string `json:"name"`
	Value Value  `json:"value,omitempty"`
}

func Attribute(name string, value interface{}) Condition {
	var s Value
	if v, ok := value.(Value); ok {
		s = v
	} else {
		s = GenericValue(value)
	}
	return newCondition("attribute", &AttributeType{
		Name:  name,
		Value: s,
	})
}

func (e *AttributeType) Create(r Registry) (logging.Condition, error) {
	if e.Name == "" {
		return nil, fmt.Errorf("attribute name missing")
	}
	v, err := r.CreateValueFromElement(&e.Value)
	if err != nil {
		return nil, fmt.Errorf("cannot parse value: %s", err)
	}
	return logging.NewAttribute(e.Name, v), nil
}

////////////////////////////////////////////////////////////////////////////////
