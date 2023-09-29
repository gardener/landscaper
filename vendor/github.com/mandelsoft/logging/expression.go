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

// And provides an AND condition for the given conditions.
func And(conditions ...Condition) Condition {
	return &AndExpr{conditions}
}

type AndExpr struct {
	conditions []Condition
}

func (e *AndExpr) Match(messageContext ...MessageContext) bool {
	for _, c := range e.conditions {
		if !c.Match(messageContext...) {
			return false
		}
	}
	return true
}

func (e *AndExpr) Conditions() []Condition {
	return sliceCopy(e.conditions)
}

// Or provides an OR condition for the given conditions.
func Or(conditions ...Condition) Condition {
	return &OrExpr{conditions}
}

type OrExpr struct {
	conditions []Condition
}

func (e *OrExpr) Match(messageContext ...MessageContext) bool {
	for _, c := range e.conditions {
		if c.Match(messageContext...) {
			return true
		}
	}
	return false
}

func (e *OrExpr) Conditions() []Condition {
	return sliceCopy(e.conditions)
}

// Not provides a NOT condition for the given condition.
func Not(condition Condition) Condition {
	return &NotExpr{condition}
}

type NotExpr struct {
	condition Condition
}

func (e *NotExpr) Match(messageContext ...MessageContext) bool {
	return !e.condition.Match(messageContext...)
}

func (e *NotExpr) Condition() Condition {
	return e.condition
}
