// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"sync"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"
)

type CleanupActionHandle *int
type cleanupAction struct {
	handle CleanupActionHandle
	action func()
}

// Cleanup contains a list of Cleanup hocks
type Cleanup struct {
	mux        sync.Mutex
	actionList []cleanupAction
}

// Add adds a Cleanup action function
func (c *Cleanup) Add(fn func()) CleanupActionHandle {
	p := CleanupActionHandle(new(int))
	c.mux.Lock()
	defer c.mux.Unlock()
	handle := cleanupAction{
		handle: p,
		action: fn,
	}
	c.actionList = append(c.actionList, handle)
	return p
}

// Remove removes a Cleanup action with the given handle from the list.
func (c *Cleanup) Remove(handle CleanupActionHandle) {
	c.mux.Lock()
	defer c.mux.Unlock()
	for i, action := range c.actionList {
		if action.handle == handle {
			c.actionList = append(c.actionList[:i], c.actionList[i+1:]...)
			return
		}
	}
}

// Run runs all functions installed by AddCleanupAction.  It does
// not remove them (see RemoveCleanupAction) but it does run unlocked, so they
// may remove themselves.
func (c *Cleanup) Run(logger utils.Logger, testsFailed bool) {
	if testsFailed {
		logger.Logln("cleanup skipped due to failed tests")
		return
	}

	list := []func(){}
	func() {
		c.mux.Lock()
		defer c.mux.Unlock()
		for _, p := range c.actionList {
			list = append(list, p.action)
		}
	}()
	// Run unlocked.
	for _, fn := range list {
		fn()
	}
}
