// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package refmgmt provides a simple wrapper, which can be used
// to map a closable object type into an interface supporting reference
// counting and supporting a Dup() method.
//
// The caller must provide an appropriate interface with the
// base object methods and additionally the Dup() (O, error) interface.
// Additionally, a struct type must be implemented hosting a View object
// (which implements the View.Close() and View.Dup() method) and
// the base object. It must implement the other interface methods
// by forwarding them to the base object.
// To create such a view object an appropriate creator function has to
// be provided.
//
// The following example illustrates the usage:
//
//	 // Objectbase is the base interface for the
//	 // object type to be wrapped.
//	 type ObjectBase interface {
//	 	io.Closer
//
//	 	Value() (string, error)
//	 }
//
//	 ////////////////////////////////////////////////////////////////////////////////
//
//	 // Object is the final user facing interface.
//	 // It includes the base interface plus the Dup method.
//	 type Object interface {
//	 	ObjectBase
//			Dup() (Object, error)
//	 }
//
//	 ////////////////////////////////////////////////////////////////////////////////
//
//	 // object is the implementation type for the bse object.
//	 type object struct {
//	 	lock   sync.Mutex
//			closed bool
//	 	value  string
//	 }
//
//	 func (o *object) Value() (string, error) {
//			if o.closed {
//	 		return "", fmt.Errorf("should not happen")
//			}
//	 	return o.value, nil
//	 }
//
//	 func (o *object) Close() error {
//			o.lock.Lock()
//			defer o.lock.Unlock()
//
//			if o.closed {
//		 		return refmgmt.ErrClosed
//			}
//			o.closed = true
//			return nil
//	 }
//
//	 ////////////////////////////////////////////////////////////////////////////////
//
//	 // view is the view object used to wrap the base object.
//	 // It forwards all methods to the base object using the
//	 // Execute function of the manager, to assure execution
//	 // on non-closed views, only.
//	 type view struct {
//			*refmgmt.View[Object]
//			obj ObjectBase
//	 }
//
//	 func (v *view) Value() (string, error) {
//			value := ""
//
//			err := v.Execute(func() (err error) {
//		 		value, err = v.obj.Value() // forward to viewd object
//		 		return
//			})
//			return value, err
//	 }
//
//	 // creator is the view object creator based on
//	 // the base object and the view manager.
//	 func creator(obj ObjectBase, v *refmgmt.View[Object]) Object {
//	 	return &view{v, obj}
//	 }
package refmgmt
