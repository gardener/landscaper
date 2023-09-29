// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"io"

	"github.com/open-component-model/ocm/pkg/common/accessio"
)

type CloserView interface {
	Close() error
	IsClosed() bool
	Execute(func() error) error
	Lazy()
}

var _ CloserView = accessio.CloserView(nil)

var ErrClosed = accessio.ErrClosed

// resourceViewInterface is a helper type used to implement parameter type
// recursion for ResourceView[T ResourceView[T]], which is not allowed in Go.
type resourceViewInterface[T any] interface {
	io.Closer
	IsClosed() bool
	Dup[T]
}

// ResourceView is the view related part of a resource interface T.
// T must incorporate ResourceView[T], which cannot directly be expressed
// in go, but with the helper interface defining the API.
type ResourceView[T resourceViewInterface[T]] interface {
	resourceViewInterface[T]
}

// ResourceViewInt can be used to execute an operation on a non-closed
// view.
type ResourceViewInt[T resourceViewInterface[T]] interface {
	resourceViewInterface[T]
	// Execute call a synchronized function on a non-closed view
	Execute(func() error) error
	Lazy()
}

type Dup[T any] interface {
	Dup() (T, error)
}

////////////////////////////////////////////////////////////////////////////////

// ViewManager is the interface of the reference manager, which
// can be used to gain new views to a managed resource.
type ViewManager[T any] interface {
	View(main ...bool) (T, error)
	IsClosed() bool
}

// ResourceViewCreator is a function which must be provided by the resource provider
// to map an implementation to the resource interface T.
// It must use NewView to create the view related part of a resource.
type ResourceViewCreator[T any, I io.Closer] func(I, CloserView, ViewManager[T]) T

type viewManager[T any, I ResourceImplementation[T]] struct {
	refs    accessio.ReferencableCloser
	creator ResourceViewCreator[T, I]
	impl    I
}

// ResourceImplementation is the minimal interface for an implementation
// a resource with managed views.
type ResourceImplementation[T any] interface {
	io.Closer
	SetViewManager(m ViewManager[T])
	ViewManager[T]
}

// NewResource creates a resource based on an implementation and a ResourceViewCreator.
// function.
func NewResource[T any, I ResourceImplementation[T]](impl I, c ResourceViewCreator[T, I], name string, main ...bool) T {
	i := &viewManager[T, I]{
		refs:    accessio.NewRefCloser(impl, true).WithName(name),
		creator: c,
		impl:    impl,
	}
	impl.SetViewManager(i)
	t, _ := i.View(main...)
	return t
}

func (i *viewManager[T, I]) View(main ...bool) (T, error) {
	var _nil T

	v, err := i.refs.View(main...)
	if err != nil {
		return _nil, err
	}
	return i.creator(i.impl, v, i), nil
}

func (i *viewManager[T, I]) IsClosed() bool {
	return i.refs.IsClosed()
}

////////////////////////////////////////////////////////////////////////////////

// noneRefCloser is used to compose a non-referencing
// view, which does not forward the close operation
// to the view manager. Its state directly reflects
// the state of the view manager.
type noneRefCloser[T io.Closer] struct {
	mgr ViewManager[T]
}

var _ CloserView = (*noneRefCloser[io.Closer])(nil)

func (n *noneRefCloser[T]) Close() error {
	if n.mgr.IsClosed() {
		return ErrClosed
	}
	return nil
}

func (n *noneRefCloser[T]) IsClosed() bool {
	return n.mgr.IsClosed()
}

func (n *noneRefCloser[T]) Execute(f func() error) error {
	v, err := n.mgr.View()
	if err != nil {
		return err
	}
	defer v.Close()
	return f()
}

func (n *noneRefCloser[T]) Lazy() {
}

////////////////////////////////////////////////////////////////////////////////

type resourceView[T any] struct {
	view CloserView
	mgr  ViewManager[T]
}

// NewView is to be called by a resource view creator to map
// the given resource implementation to complete resource interface.
// It should create an object with two local embedded fields:
//   - the returned ResourceView and the
//   - given resource implementation.
func NewView[T resourceViewInterface[T]](v CloserView, d ViewManager[T]) ResourceViewInt[T] {
	return &resourceView[T]{v, d}
}

// NewNonRefView provides a reference-less view directly for the reference manager.
// It is valid as long as the reference manager is not closed with the last regular
// reference.
func NewNonRefView[T resourceViewInterface[T]](d ViewManager[T]) ResourceViewInt[T] {
	return &resourceView[T]{&noneRefCloser[T]{d}, d}
}

func NoneRefCloserView[T io.Closer](d ViewManager[T]) CloserView {
	return &noneRefCloser[T]{d}
}

func (v *resourceView[T]) IsClosed() bool {
	return v.view.IsClosed()
}

func (v *resourceView[T]) Close() error {
	return v.view.Close()
}

func (v *resourceView[T]) Execute(f func() error) error {
	return v.view.Execute(f)
}

func (v *resourceView[T]) Dup() (t T, err error) {
	err = v.Execute(func() error {
		t, err = v.mgr.View()
		return err
	})
	return t, err
}

func (v *resourceView[T]) Lazy() {
	v.view.Lazy()
}

////////////////////////////////////////////////////////////////////////////////

type ResourceImplBase[T any] struct {
	refs   ViewManager[T]
	closer []io.Closer
}

// NewResourceImplBase creates an implementation base for a resource T
// referencing another resource M.
func NewResourceImplBase[T any, M io.Closer](m ViewManager[M], closer ...io.Closer) (*ResourceImplBase[T], error) {
	if m != nil {
		ref, err := m.View()
		if err != nil {
			return nil, err
		}
		closer = append(closer, ref)
	}
	return &ResourceImplBase[T]{
		closer: closer,
	}, nil
}

func (b *ResourceImplBase[T]) SetViewManager(m ViewManager[T]) {
	b.refs = m
}

func (b *ResourceImplBase[T]) View(main ...bool) (T, error) {
	return b.refs.View(main...)
}

func (b *ResourceImplBase[T]) IsClosed() bool {
	return b.refs.IsClosed()
}

func (b *ResourceImplBase[T]) Close() error {
	return accessio.Close(b.closer...)
}
