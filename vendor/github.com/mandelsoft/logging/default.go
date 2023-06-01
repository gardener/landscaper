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
	"sync/atomic"
)

var defaultContext *ContextReference
var seq atomic.Value

type ContextId int64

func init() {
	seq.Store(ContextId(0))
	defaultContext = &ContextReference{NewDefault()}
}

func getId() ContextId {
	for {
		o := seq.Load().(ContextId)
		if seq.CompareAndSwap(o, o+1) {
			return o + 1
		}
	}
}

// SetDefaultContext sets the default context.
// It changes all usages based on the result of
// DefaultContext().
// If the rules of the actual default context should be kept
// use it as base context or copy the rules with
// DefaultContext().AddTo(newContext) before setting the new
// context as default context (potentially unsafe because of race condition).
func SetDefaultContext(ctx Context) {
	defaultContext.Context = ctx
}

// DefaultContext returns a context, which always reflects
// the actually set default context.
func DefaultContext() Context {
	return defaultContext
}

func Log(messageContext ...MessageContext) Logger {
	return defaultContext.Logger(messageContext...)
}

type ContextReference struct {
	Context
}
