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
	"sync"
	"sync/atomic"
)

// UpdateState remembers the config level of a root logging context.
type UpdateState struct {
	generation atomic.Value
}

// Next provides the next generation number of a context tree.
func (s *UpdateState) Next() int64 {
	for {
		old := s.generation.Load()

		new := int64(1)
		if old != nil {
			new = old.(int64) + 1
		}
		if s.generation.CompareAndSwap(old, new) {
			return new
		}
	}
}

// Generation returns the actual generation number of a context tree.
func (s *UpdateState) Generation() int64 {
	cur := s.generation.Load()
	if cur == nil {
		return 0
	}
	return cur.(int64)
}

// Updater is used by a logging context to check for new updates
// in a context tree.
type Updater struct {
	lock      sync.Mutex
	state     *UpdateState
	base      *Updater
	watermark int64
	seen      int64
}

func NewUpdater(base *Updater) *Updater {
	u := &Updater{base: base}
	if base == nil {
		u.state = &UpdateState{}
	} else {
		u.state = base.state
		u.seen = base.SeenWatermark()
		u.watermark = base.Watermark()
	}
	return u
}

func (u *Updater) Modify() {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.watermark = u.state.Next()
	if u.base == nil {
		u.seen = u.watermark
	}
}

func (u *Updater) Watermark() int64 {
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.base != nil {
		w := u.base.Watermark()
		if w > u.watermark {
			u.watermark = w
		}
	}
	return u.watermark
}

func (u *Updater) SeenWatermark() int64 {
	u.lock.Lock()
	defer u.lock.Unlock()
	return u.seen
}

// Require returns whether a local config update is required.
func (u *Updater) Require() bool {
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.base == nil {
		return false
	}
	w := u.base.Watermark()
	if w > u.seen {
		u.seen = w
		return true
	}
	return false
}
