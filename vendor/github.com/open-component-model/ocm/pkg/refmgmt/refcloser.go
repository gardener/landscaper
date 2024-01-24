// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package refmgmt

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/open-component-model/ocm/pkg/errors"
)

var ErrClosed = errors.ErrClosed()

// ReferencableCloser manages closable views to a basic closer.
// If the last view is closed, the basic closer is finally closed.
type ReferencableCloser interface {
	ExtendedAllocatable

	RefCount() int
	UnrefLast() error
	IsClosed() bool

	Closer() io.Closer
	View(main ...bool) (CloserView, error)

	WithName(name string) ReferencableCloser
}

type referencableCloser struct {
	RefMgmt
	closer io.Closer
}

func NewRefCloser(closer io.Closer, unused ...bool) ReferencableCloser {
	return &referencableCloser{RefMgmt: NewAllocatable(closer.Close, unused...), closer: closer}
}

func (r *referencableCloser) WithName(name string) ReferencableCloser {
	r.RefMgmt.WithName(name)
	return r
}

func (r *referencableCloser) Closer() io.Closer {
	return r.closer
}

// View creates a new closable view.
// The object is closed if the last view has been released.
// If main is set to true, the close method of the view
// returns an error, if it is not the last reference.
func (r *referencableCloser) View(main ...bool) (CloserView, error) {
	err := r.Ref()
	if err != nil {
		return nil, err
	}
	v := &view{ref: r}
	for _, b := range main {
		if b {
			v.main = true
		}
	}
	return v, nil
}

type LazyMode interface {
	Lazy()
}

type RefCountProvider interface {
	RefCount() int
}

// ToLazy resets the main view flag
// of closer views to enable
// dark release of resources even if the
// first/main view has been closed.
// Otherwise, closing the main view will
// fail, if there are still subsequent views.
func ToLazy[T any](o T, err error) (T, error) {
	if err == nil {
		Lazy(o)
	}
	return o, err
}

func AsLazy[T any](o T) T {
	Lazy(o)
	return o
}

func Lazy(o interface{}) bool {
	if o == nil {
		return false
	}
	if l, ok := o.(LazyMode); ok {
		l.Lazy()
		return true
	}
	return false
}

func ReferenceCount(o interface{}) int {
	if o != nil {
		if l, ok := o.(RefCountProvider); ok {
			return l.RefCount()
		}
	}
	return -1
}

func CloseTemporary(c io.Closer) error {
	if !Lazy(c) {
		return errors.ErrNotSupported("lazy mode")
	}
	AllocLog.Trace("close temporary ref")
	return c.Close()
}

func PropagateCloseTemporary(errp *error, c io.Closer) {
	errors.PropagateError(errp, func() error { return CloseTemporary(c) })
}

type CloserView interface {
	io.Closer
	LazyMode
	RefCountProvider

	IsClosed() bool

	View() (CloserView, error)

	Release() error
	Finalize() error

	Closer() io.Closer

	Execute(f func() error) error
	Allocatable() ExtendedAllocatable
}

type view struct {
	lock   sync.Mutex
	ref    ReferencableCloser
	main   bool
	closed atomic.Bool
}

var _ CloserView = (*view)(nil)

func (v *view) Lazy() {
	v.main = false
}

func (v *view) RefCount() int {
	return v.ref.RefCount()
}

func (v *view) Allocatable() ExtendedAllocatable {
	return v.ref
}

func (v *view) Execute(f func() error) error {
	v.lock.Lock()
	defer v.lock.Unlock()
	if v.closed.Load() {
		return ErrClosed
	}
	return f()
}

// Release will release the view.
// With releasing the last view
// the underlying object will be closed.
func (v *view) Release() error {
	v.lock.Lock()
	defer v.lock.Unlock()
	if v.closed.Load() {
		return ErrClosed
	}
	err := v.ref.Unref()
	v.closed.Store(true)
	return err
}

// Finalize will try to finalize the
// underlying object. This is only
// possible if no further view is
// still pending.
func (v *view) Finalize() error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.closed.Load() {
		return ErrClosed
	}

	if err := v.ref.UnrefLast(); err != nil {
		return errors.Wrapf(err, "unable to unref last")
	}
	v.closed.Store(true)
	return nil
}

func (v *view) Close() error {
	if v.main {
		return v.Finalize()
	}

	return v.Release()
}

func (v *view) IsClosed() bool {
	return v.closed.Load()
}

func (v *view) View() (CloserView, error) {
	return v.ref.View()
}

func (v *view) Closer() io.Closer {
	return v.ref.Closer()
}

type Closers []io.Closer

func (c *Closers) Add(closers ...io.Closer) {
	for _, e := range closers {
		if e != nil {
			*c = append(*c, e)
		}
	}
}

func (c Closers) Effective() io.Closer {
	switch len(c) {
	case 0:
		return nil
	case 1:
		return c[0]
	default:
		return c
	}
}

func (c Closers) Close() error {
	list := errors.ErrList()
	for _, e := range c {
		list.Add(e.Close())
	}
	return list.Result()
}

type CloserFunc func() error

func (c CloserFunc) Close() error {
	return c()
}
