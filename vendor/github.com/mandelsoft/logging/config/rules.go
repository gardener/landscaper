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
	"github.com/mandelsoft/logging"
	"github.com/mandelsoft/logging/scheme"
)

func init() {
	RegisterRule("rule", &ConditionalRuleType{})
}

func newRule(typ string, v RuleType) Rule {
	return scheme.NewElement(typ, v)
}

type ConditionalRuleType struct {
	Level      string      `json:"level"`
	Conditions []Condition `json:"conditions"`
}

func ConditionalRule(level string, conds ...Condition) Rule {
	return newRule("rule", &ConditionalRuleType{level, conds})
}

func (r *ConditionalRuleType) Create(reg Registry) (logging.Rule, error) {
	l, err := logging.ParseLevel(r.Level)
	if err != nil {
		return nil, err
	}
	conditions, err := ParseConditions(reg, r.Conditions)
	if err != nil {
		return nil, err
	}
	return logging.NewConditionRule(l, conditions...), nil
}
