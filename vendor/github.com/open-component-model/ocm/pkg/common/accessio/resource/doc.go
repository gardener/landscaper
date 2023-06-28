// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package resource provides support to implement
// closeable backing resources featuring multiple
// separately closeable references. The backing resource
// is finally closed, when the last reference is closed.
// The close method can then be used, for example, to release
// temporary external resources with the last released reference.
// Hereby, the reference implements the intended resource
// interface including the reference related part, which
// includes a Dup method, which can be used to gain a
// new additional reference to the backing object.
//
// Those references are called Views in the package.
// The backing object implements the pure resource
// object interface plus the final Close method.
//
// The final resource interface is described by a Go
// interface including the resource.ResourceView interface,
//
//	type MyResource interface {
//	   resource.ResourceView[MyResource]
//	   AdditionalMethods()...
//	}
//
// The resource.ResourceView interface offers the view-related
// methods.
//
// With NewResource a new view management and a first
// view is created for this object. This method is typically
// wrapped by a dedicated resource creator function:
//
//	func New(args...) MyResource {
//	   i := MyResourceImpl{
//	          ...
//	        }
//	   return resource.NewResource(i, myViewCreator)
//	}
//
// The interface ResourceImplementation describes the minimal
// interface an implementation object has to implement to
// work with this view management package.
// It gets access to the ViewManager to be able to
// create new views/references for potential sub objects
// provided by the implementation, which need access to
// the implementation. In such a case those sub objects
// require a Close method again, are may even use an
// own view management.
//
// The management as well as the view can be used to create
// additional views.
//
// Therefore, the reference management uses a ResourceViewCreator
// function, which must be provided by the object implementation
// Its task is to create a new frontend view object implementing
// the desired pure backing object functionality plus the
// view-related interface.
//
// This is done by creating an object with two embedded fields:
//
//	type MyReference struct {
//	   resource.ReferenceView[MyInterface]
//	   MyImplementation
//	}
//
// the myViewCreator function creates a new resource reference using the
// resource.NewView function.
//
//	func myViewCreator(impl *ResourceImpl,
//	                   v resource.CloserView,
//	                   d resource.Dup[Resource]) MyResource {
//	  return &MyResource {
//	           resource.NewView(v, d),
//	           impl,
//	         }
//	}
//
// A default resource base implementation is provided by ResourceImplBase.
// It implements the minimal implementation interface and offers with
// the method View a way to create additional views. It can just be
// instantiated for the base usage.
// Using the creator NewResourceImplBase is is possible to support
//   - nested use-cases, where an implementations hold a reference
//     on a parent object
//   - additional closers are required.
//
// Therefore, it provides a default Close method. If your implementation
// required an additional cleanup, you have to reimplement the Close
// method and call at least the base implementation method. Or you
// configure the optional closer for the base implementation.
package resource
