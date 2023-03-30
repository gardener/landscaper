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

// UpdateState remembers the config level of a root logging context.
type UpdateState struct {
	level atomic.Value
}

// Modify triggers the notification of an update of an element of
// a context tree.
func (s *UpdateState) Modify() {
	for {
		old := s.level.Load()

		new := int64(1)
		if old != nil {
			new = old.(int64) + 1
		}
		if s.level.CompareAndSwap(old, new) {
			return
		}
	}
}

// Level returns the actual update level of context tree.
func (s *UpdateState) Level() int64 {
	cur := s.level.Load()
	if cur == nil {
		return 0
	}
	return cur.(int64)
}

// Updater is used by a logging context to check for new updates
// in a context tree.
type Updater struct {
	state     *UpdateState
	watermark int64
}

func NewUpdater(s *UpdateState) *Updater {
	return &Updater{state: s}
}

func (u *Updater) Watermark() int64 {
	return u.watermark
}

// Require returns whether a config update is potentially required.
func (u *Updater) Require() bool {
	level := u.state.Level()
	if level > u.watermark {
		u.watermark = level
		return true
	}
	return false
}
