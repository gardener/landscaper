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

package logging

import (
	"reflect"
)

type ConditionRule struct {
	conditions []Condition
	level      int
}

var _ Rule = (*ConditionRule)(nil)
var _ UpdatableRule = (*ConditionRule)(nil)

func NewConditionRule(level int, cond ...Condition) Rule {
	return &ConditionRule{
		level:      level,
		conditions: cond,
	}
}

func (r *ConditionRule) MatchRule(o Rule) bool {
	if or, ok := o.(*ConditionRule); ok {
		return reflect.DeepEqual(r.conditions, or.conditions)
	}
	return false
}

func (r *ConditionRule) Match(sink SinkFunc, messageContext ...MessageContext) Logger {
	for _, c := range r.conditions {
		if !c.Match(messageContext...) {
			return nil
		}
	}

	return NewLogger(DynSink(AsLevelFunc(r.level), 0, sink))
}

func (r *ConditionRule) Level() int {
	return r.level
}

func (r *ConditionRule) Conditions() []Condition {
	return sliceCopy(r.conditions)
}
