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
	"github.com/mandelsoft/logging/scheme"
)

////////////////////////////////////////////////////////////////////////////////

func newValue(typ string, v ValueType) Value {
	return scheme.NewElement(typ, v)
}

type GenericValueType struct {
	Value interface{}
}

// MarshalJSON returns m as the JSON encoding of m.
func (m GenericValueType) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Value)
}

// UnmarshalJSON sets *m to a copy of data.
func (m *GenericValueType) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.Value)
}

func (m GenericValueType) Create(r Registry) (interface{}, error) {
	return m.Value, nil
}

func GenericValue(v interface{}) Value {
	s := GenericValueType{v}
	return newValue("value", &s)
}

type Rule = scheme.Element[RuleType]
type Condition = scheme.Element[ConditionType]
type Value = scheme.Element[ValueType]

type RuleType = scheme.Factory[logging.Rule, Registry]
type ConditionType = scheme.Factory[logging.Condition, Registry]
type ValueType = scheme.Factory[any, Registry]

type Registry interface {
	RegisterRuleType(name string, ty RuleType)
	RegisterConditionType(name string, ty ConditionType)
	RegisterValueType(name string, ty ValueType)

	CreateConditionFromElement(e *Condition) (logging.Condition, error)
	CreateRuleFromElement(e *Rule) (logging.Rule, error)
	CreateValueFromElement(e *Value) (any, error)
	CreateCondition(data []byte) (logging.Condition, error)
	CreateRule(data []byte) (logging.Rule, error)
	CreateValue(data []byte) (any, error)

	Evaluate(cfg *Config) error
	EvaluateFromData(data []byte) (*Config, error)

	Configure(ctx logging.Context, cfg *Config) error
	ConfigureWithData(ctx logging.Context, data []byte) error

	Copy() Registry
}

type registry struct {
	rules      *scheme.Scheme[logging.Rule, Registry]
	conditions *scheme.Scheme[logging.Condition, Registry]
	values     *scheme.Scheme[any, Registry]
}

func NewRegistry() Registry {
	r := &registry{}
	r.rules = scheme.NewScheme[logging.Rule, Registry](r)
	r.conditions = scheme.NewScheme[logging.Condition, Registry](r)
	r.values = scheme.NewScheme[interface{}, Registry](r)
	r.RegisterValueType("value", GenericValueType{})
	return r
}

func (r *registry) Copy() Registry {
	c := &registry{}
	c.rules = r.rules.Copy(c)
	c.conditions = r.conditions.Copy(c)
	c.values = r.values.Copy(c)
	return c
}

func (r *registry) RegisterRuleType(name string, ty RuleType) {
	r.rules.Register(name, ty)
}

func (r *registry) RegisterConditionType(name string, ty ConditionType) {
	r.conditions.Register(name, ty)
}

func (r *registry) RegisterValueType(name string, ty ValueType) {
	r.values.Register(name, ty)
}

func (r *registry) CreateRuleFromElement(e *Rule) (logging.Rule, error) {
	return r.rules.GetFromElement(e)
}

func (r *registry) CreateConditionFromElement(e *Condition) (logging.Condition, error) {
	return r.conditions.GetFromElement(e)
}

func (r *registry) CreateValueFromElement(e *Value) (interface{}, error) {
	return r.values.GetFromElement(e)
}

func (r *registry) CreateRule(data []byte) (logging.Rule, error) {
	return r.rules.Get(data)
}

func (r *registry) CreateCondition(data []byte) (logging.Condition, error) {
	return r.conditions.Get(data)
}

func (r *registry) CreateValue(data []byte) (interface{}, error) {
	return r.values.Get(data)
}

func (r *registry) EvaluateFromData(data []byte) (*Config, error) {
	var cfg Config

	err := cfg.UnmarshalFrom(data)
	if err != nil {
		return nil, err
	}

	err = r.Evaluate(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *registry) Evaluate(cfg *Config) error {
	if cfg.DefaultLevel != "" {
		_, err := logging.ParseLevel(cfg.DefaultLevel)
		if err != nil {
			return fmt.Errorf("default level: %w", err)
		}
	}

	for i := range cfg.Rules {
		_, err := r.CreateRuleFromElement(&cfg.Rules[i])
		if err != nil {
			return fmt.Errorf("cannot parse rule %d: %w", i, err)
		}
	}
	return nil
}

func (r *registry) Configure(ctx logging.Context, cfg *Config) error {
	if cfg.DefaultLevel != "" {
		l, err := logging.ParseLevel(cfg.DefaultLevel)
		if err != nil {
			return fmt.Errorf("default level: %w", err)
		}
		ctx.SetDefaultLevel(l)
	}

	for i := range cfg.Rules {
		rule, err := r.CreateRuleFromElement(&cfg.Rules[i])
		if err != nil {
			return fmt.Errorf("cannot parse rule %d: %w", i, err)
		}
		ctx.AddRule(rule)
	}
	return nil
}

func (r *registry) ConfigureWithData(ctx logging.Context, data []byte) error {
	var cfg Config

	err := cfg.UnmarshalFrom(data)
	if err != nil {
		return err
	}

	return r.Configure(ctx, &cfg)
}

func ParseConditions(r Registry, list []Condition) ([]logging.Condition, error) {
	conditions := []logging.Condition{}
	for i := range list {
		c, err := r.CreateConditionFromElement(&list[i])
		if err != nil {
			return nil, fmt.Errorf("cannot parse condition %d: %w", i, err)
		}
		conditions = append(conditions, c)
	}
	return conditions, nil
}

var _registry = NewRegistry()

func DefaultRegistry() Registry {
	return _registry
}

func RegisterRule(name string, ty RuleType) {
	_registry.RegisterRuleType(name, ty)
}

func RegisterCondition(name string, ty ConditionType) {
	_registry.RegisterConditionType(name, ty)
}

func RegisterValueType(name string, ty ValueType) {
	_registry.RegisterValueType(name, ty)
}
