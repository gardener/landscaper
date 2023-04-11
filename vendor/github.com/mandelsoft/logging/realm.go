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

type realm struct {
	name   string
	prefix bool
}

var _ Realm = (*realm)(nil)

// DefineRealm creates a tag and registers it together with a description.
func DefineRealm(name string, desc string) Realm {
	defs.DefineRealm(name, desc)
	return NewRealm(name)
}

// NewRealm provides a new Realm object to be used as rule condition
// or message context.
func NewRealm(name string) Realm {
	return &realm{name: name}
}

// NewRealmPrefix provides a new Realm object to be used as rule condition
// matching a realm prefix.
func NewRealmPrefix(name string) RealmPrefix {
	return &realm{name: name, prefix: true}
}

func (r *realm) Match(messageContext ...MessageContext) bool {
	for _, c := range messageContext {
		if e, ok := c.(Realm); ok && r.check(e.Name()) {
			return true
		}
	}
	return false
}

func (r *realm) check(name string) bool {
	if name == r.name {
		return true
	}
	return r.IsPrefix() && strings.HasPrefix(name, r.name+"/")
}

func (r *realm) Name() string {
	return r.name
}

func (r *realm) IsPrefix() bool {
	return r.prefix
}

func (r *realm) Attach(l Logger) Logger {
	if r.IsPrefix() {
		return l
	}
	return l.WithName(r.name)
}

func (r *realm) String() string {
	return r.name
}

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
