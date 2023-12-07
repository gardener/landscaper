// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package refmgmt

import (
	"io"
)

// Dup is the common interface for all
// objects following the ref counting model.
// It will provide a new view to the underlying
// object. This will only be closed if there are
// no more views (created via Dup()).
type Dup[V io.Closer] interface {
	Dup() (V, error)
}

type ViewManager[V io.Closer] interface {
	View(closerView CloserView) (V, error)
}

type viewManager[O, V io.Closer /* O */] struct {
	refs    ReferencableCloser
	obj     O
	creator func(O, *View[V]) V
}

func WithView[O, V io.Closer](obj O, creator func(O, *View[V]) V, closer ...io.Closer) V {
	var c Closers

	c.Add(obj)
	c.Add(closer...)
	m := &viewManager[O, V]{
		refs:    NewRefCloser(c.Effective(), true),
		obj:     obj,
		creator: creator,
	}
	v, _ := m.View(nil)
	return v
}

func (m *viewManager[O, V]) View(v CloserView) (V, error) {
	if v != nil {
		var n V
		err := v.Execute(func() (err error) {
			n, err = m.view()
			return
		})
		return n, err
	}
	return m.view()
}

func (m *viewManager[O, V]) view() (V, error) {
	var _nil V

	v, err := m.refs.View(false)
	if err != nil {
		return _nil, err
	}

	return m.creator(m.obj, NewView[V](m, v)), nil
}

type View[V io.Closer] struct {
	mgr  ViewManager[V]
	view CloserView
}

var _ Dup[io.Closer] = (*View[io.Closer])(nil)

func NewView[V io.Closer](mgr ViewManager[V], v CloserView) *View[V] {
	return &View[V]{mgr: mgr, view: v}
}

func (v *View[V]) Dup() (V, error) {
	return v.mgr.View(v.view)
}

func (v *View[V]) Close() error {
	return v.view.Close()
}

func (v *View[V]) IsClosed() bool {
	return v.view.IsClosed()
}

func (v *View[V]) Execute(f func() error) error {
	return v.view.Execute(f)
}
