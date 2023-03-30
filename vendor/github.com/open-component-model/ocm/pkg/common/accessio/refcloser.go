// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"fmt"
	"io"
	"sync"
)

// ReferencableCloser manages closable views to a basic closer.
// If the last view is closed, the basic closer is finally closed.
type ReferencableCloser interface {
	RefMgmt

	Closer() io.Closer
	View(main ...bool) (CloserView, error)
}

type referencableCloser struct {
	RefMgmt
	closer io.Closer
}

func NewRefCloser(closer io.Closer, unused ...bool) ReferencableCloser {
	return &referencableCloser{RefMgmt: NewAllocatable(closer.Close, unused...), closer: closer}
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

type CloserView interface {
	io.Closer
	IsClosed() bool

	View() (CloserView, error)

	Release() error
	Finalize() error

	Closer() io.Closer

	Execute(f func() error) error
}

type view struct {
	lock   sync.Mutex
	ref    ReferencableCloser
	main   bool
	closed bool
}

var _ CloserView = (*view)(nil)

func (v *view) Execute(f func() error) error {
	v.lock.Lock()
	defer v.lock.Unlock()
	if v.closed {
		return ErrClosed
	}
	return f()
}

func (v *view) Release() error {
	v.lock.Lock()
	defer v.lock.Unlock()
	if v.closed {
		return ErrClosed
	}
	v.closed = true
	return v.ref.Unref()
}

func (v *view) Finalize() error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.closed {
		return ErrClosed
	}

	if err := v.ref.UnrefLast(); err != nil {
		return fmt.Errorf("unable to unref last: %w", err)
	}

	v.closed = true

	return nil
}

func (v *view) Close() error {
	if v.main {
		return v.Finalize()
	}

	return v.Release()
}

func (v *view) IsClosed() bool {
	v.lock.Lock()
	defer v.lock.Unlock()

	return v.closed
}

func (v *view) View() (CloserView, error) {
	return v.ref.View()
}

func (v *view) Closer() io.Closer {
	return v.ref.Closer()
}
