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
	"runtime"
	"strings"
)

type realm string

// DefineRealm creates a tag and registers it together with a description.
func DefineRealm(name string, desc string) Realm {
	defs.DefineRealm(name, desc)
	return NewRealm(name)
}

// NewRealm provides a new Realm object to be used as rule condition
// or message context.
func NewRealm(name string) Realm {
	return realm(name)
}

func (r realm) Name() string {
	return string(r)
}

func (r realm) Match(messageContext ...MessageContext) bool {
	return matchRealm(string(r), false, messageContext...)
}

func (r realm) Attach(l Logger) Logger {
	return l.WithValues(FieldKeyRealm, string(r))
}

////////////////////////////////////////////////////////////////////////////////

type realmprefix string

// NewRealmPrefix provides a new RealmPrefix object to be used as rule condition
// matching a realm prefix.
func NewRealmPrefix(name string) RealmPrefix {
	return realmprefix(name)
}

func (r realmprefix) Name() string {
	return string(r)
}

func (r realmprefix) Match(messageContext ...MessageContext) bool {
	return matchRealm(string(r), true, messageContext...)
}

////////////////////////////////////////////////////////////////////////////////

func matchRealm(r string, prefix bool, messageContext ...MessageContext) bool {
	// match only last (most significant) realm in complete aggregated
	// message context
	for i := len(messageContext) - 1; i >= 0; i-- {
		if e, ok := messageContext[i].(Realm); ok {
			return checkRealm(r, e.Name(), prefix)
		}
	}
	return false
}

func checkRealm(r, name string, prefix bool) bool {
	if name == r {
		return true
	}
	return prefix && strings.HasPrefix(name, r+"/")
}

////////////////////////////////////////////////////////////////////////////////

func Package() Realm {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return NewRealm("<unknown>")
	}

	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	firstDot := strings.IndexByte(funcName[lastSlash:], '.') + lastSlash
	return NewRealm(funcName[:firstDot])
}
