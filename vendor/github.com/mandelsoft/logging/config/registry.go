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

type GenericValue json.RawMessage

// MarshalJSON returns m as the JSON encoding of m.
func (m GenericValue) MarshalJSON() ([]byte, error) {
	return json.RawMessage(m).MarshalJSON()
}

// UnmarshalJSON sets *m to a copy of data.
func (m *GenericValue) UnmarshalJSON(data []byte) error {
	return (*json.RawMessage)(m).UnmarshalJSON(data)
}

func (m GenericValue) Create(r Registry) (interface{}, error) {
	var v interface{}
	err := json.Unmarshal([]byte(m), &v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

type RuleType = scheme.Factory[logging.Rule, Registry]
type ConditionType = scheme.Factory[logging.Condition, Registry]
type ValueType = scheme.Factory[any, Registry]

type Registry interface {
	RegisterRuleType(name string, ty RuleType)
	RegisterConditionType(name string, ty ConditionType)
	RegisterValueType(name string, ty ValueType)

	CreateCondition(data []byte) (logging.Condition, error)
	CreateRule(data []byte) (logging.Rule, error)
	CreateValue(data []byte) (interface{}, error)

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
	r.RegisterValueType("value", GenericValue{})
	return r
}

func (r *registry) Copy() Registry {
	c := &registry{}
	c.rules = r.rules.Copy(c)
	c.conditions = r.conditions.Copy(c)
	c.values = r.values.Copy(c)
	return c
}

func (r *registry) RegisterRuleType(name string, ty scheme.Factory[logging.Rule, Registry]) {
	r.rules.Register(name, ty)
}

func (r *registry) RegisterConditionType(name string, ty scheme.Factory[logging.Condition, Registry]) {
	r.conditions.Register(name, ty)
}

func (r *registry) RegisterValueType(name string, ty scheme.Factory[any, Registry]) {
	r.values.Register(name, ty)
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

func (r *registry) Configure(ctx logging.Context, cfg *Config) error {
	if cfg.DefaultLevel != "" {
		l, err := logging.ParseLevel(cfg.DefaultLevel)
		if err != nil {
			return fmt.Errorf("default level: %w", err)
		}
		ctx.SetDefaultLevel(l)
	}

	for i, d := range cfg.Rules {
		rule, err := r.CreateRule(d)
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

func ParseConditions(r Registry, list []json.RawMessage) ([]logging.Condition, error) {
	conditions := []logging.Condition{}
	for i, d := range list {
		c, err := r.CreateCondition(d)
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
