/*
 * Copyright 2023 Mandelsoft. All rights reserved.
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
	"sort"
	"sync"
)

type Definitions map[string][]string

var defs = &definitions{
	tags:   Definitions{},
	realms: Definitions{},
}

func GetTagDefinitions() Definitions {
	return defs.GetTags()
}

func GetRealmDefinitions() Definitions {
	return defs.GetRealms()
}

type definitions struct {
	lock   sync.Mutex
	tags   map[string][]string
	realms map[string][]string
}

func (d *definitions) DefineTag(name, desc string) {
	d.define(d.tags, name, desc)
}

func (d *definitions) DefineRealm(name, desc string) {
	d.define(d.realms, name, desc)
}

func (d *definitions) define(m map[string][]string, name, desc string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	old := m[name]
	if desc != "" {
		found := false
		for _, t := range old {
			if t == desc {
				found = true
				break
			}
		}
		if !found {
			old = append(old, desc)
			sort.Strings(old)
		}
	}
	m[name] = old
}

func (d *definitions) GetTags() map[string][]string {
	d.lock.Lock()
	defer d.lock.Unlock()

	return d.tags
}

func (d *definitions) GetRealms() map[string][]string {
	d.lock.Lock()
	defer d.lock.Unlock()

	return d.realms
}
