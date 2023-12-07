// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package refmgmt

import (
	"fmt"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/logging"
)

var ALLOC_REALM = logging.DefineSubRealm("reference counting", "refcnt")

var AllocLog = logging.DynamicLogger(ALLOC_REALM)

type Allocatable interface {
	Ref() error
	Unref() error
}

type CleanupHandler interface {
	Cleanup()
}

type CleanupHandlerFunc func()

func (f CleanupHandlerFunc) Cleanup() {
	f()
}

type ExtendedAllocatable interface {
	BeforeCleanup(f CleanupHandler)
	Ref() error
	Unref() error
}

type RefMgmt interface {
	UnrefLast() error
	ExtendedAllocatable
	IsClosed() bool
	RefCount() int

	WithName(name string) RefMgmt
}

type refMgmt struct {
	lock     sync.Mutex
	refcount int
	closed   bool
	before   []CleanupHandler
	cleanup  func() error
	name     string
}

func NewAllocatable(cleanup func() error, unused ...bool) RefMgmt {
	n := 1
	for _, b := range unused {
		if b {
			n = 0
		}
	}
	return &refMgmt{refcount: n, cleanup: cleanup, name: "object"}
}

func (c *refMgmt) WithName(name string) RefMgmt {
	c.name = name
	return c
}

func (c *refMgmt) IsClosed() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.closed
}

func (c *refMgmt) Ref() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closed {
		return ErrClosed
	}
	c.refcount++
	AllocLog.Trace("ref", "name", c.name, "refcnt", c.refcount)
	return nil
}

func (c *refMgmt) Unref() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closed {
		return ErrClosed
	}

	var err error

	c.refcount--
	AllocLog.Trace("unref", "name", c.name, "refcnt", c.refcount)
	if c.refcount <= 0 {
		for _, f := range c.before {
			f.Cleanup()
		}
		if c.cleanup != nil {
			err = c.cleanup()
		}
		c.closed = true
	}

	if err != nil {
		return fmt.Errorf("unable to unref %s: %w", c.name, err)
	}

	return nil
}

func (c *refMgmt) RefCount() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.refcount
}

func (c *refMgmt) BeforeCleanup(f CleanupHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.before = append(c.before, f)
}

func (c *refMgmt) UnrefLast() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closed {
		return ErrClosed
	}

	if c.refcount > 1 {
		AllocLog.Trace("unref last failed", "name", c.name, "pending", c.refcount)
		return errors.ErrStillInUseWrap(errors.Newf("%d reference(s) pending", c.refcount), c.name)
	}

	var err error

	c.refcount--
	AllocLog.Trace("unref last", "name", c.name, "refcnt", c.refcount)
	if c.refcount <= 0 {
		for _, f := range c.before {
			f.Cleanup()
		}
		if c.cleanup != nil {
			err = c.cleanup()
		}

		c.closed = true
	}

	if err != nil {
		AllocLog.Trace("cleanup last failed", "name", c.name, "error", err.Error())
		return errors.Wrapf(err, "unable to cleanup %s while unref last", c.name)
	}

	return nil
}
