// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sync"
)

// Updater implements the generation based update protocol
// to update data contexts based on the config requests
// made to a configuration context.
type Updater interface {
	// Update replays missing configuration requests
	// applicable for a dedicated type of context or configuration target
	// stored in a configuration context.
	// It should be created for and called from within such a context
	Update() error
	State() (int64, bool)
	GetContext() Context

	Lock()
	Unlock()
	RLock()
	RUnlock()
}

type updater struct {
	sync.RWMutex
	ctx            Context
	target         interface{}
	lastGeneration int64
	inupdate       bool
}

// NewUpdater create a configuration updater for a configuration target
// based on a dedicated configuration context.
func NewUpdater(ctx Context, target interface{}) Updater {
	return &updater{
		ctx:    ctx,
		target: target,
	}
}

func (u *updater) GetContext() Context {
	return u.ctx
}

func (u *updater) GetTarget() interface{} {
	return u.target
}

func (u *updater) State() (int64, bool) {
	u.RLock()
	defer u.RUnlock()
	return u.lastGeneration, u.inupdate
}

func (u *updater) Update() error {
	u.Lock()
	if u.inupdate {
		u.Unlock()
		return nil
	}
	u.inupdate = true
	u.Unlock()

	gen, err := u.ctx.ApplyTo(u.lastGeneration, u.target)

	u.Lock()
	defer u.Unlock()
	u.inupdate = false
	u.lastGeneration = gen
	return err
}
