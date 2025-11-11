// Copyright 2021 Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package keymutex

import "sync"

type KeyMutex struct {
	mut     sync.Mutex
	mutexes map[string]*sync.Mutex
}

// New creates a new key mutex
func New() *KeyMutex {
	return &KeyMutex{
		mut:     sync.Mutex{},
		mutexes: make(map[string]*sync.Mutex),
	}
}

// Lock sets a lock on a specific key
func (km *KeyMutex) Lock(key string) {
	km.mut.Lock()
	defer km.mut.Unlock()
	mut, ok := km.mutexes[key]
	if !ok {
		km.mutexes[key] = &sync.Mutex{}
		km.mutexes[key].Lock()
		return
	}
	mut.Lock()
}

// Unlock removes a lock from a specific key
func (km *KeyMutex) Unlock(key string) {
	km.mut.Lock()
	defer km.mut.Unlock()
	if mut, ok := km.mutexes[key]; ok {
		mut.Unlock()
	}
}

// Remove removes a not needed mutex for a key.
func (km *KeyMutex) Remove(key string) {
	km.mut.Lock()
	defer km.mut.Unlock()
	delete(km.mutexes, key)
}
