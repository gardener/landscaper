// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package finalizer

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/exception"
)

type Finalizable interface {
	Finalize() error
}

// Finalizer gathers finalization functions and calls
// them by calling the Finalize method(s).
// Add and Finalize may be called in any sequence and number.
// Finalize just calls the aggregated functions between its
// last and the actual call.
// This way it can be used together with defer to clean up
// stuff when leaving a function and combine it with
// controlled intermediate cleanup needed, for example as part of
// a loop block.
type Finalizer struct {
	lock    sync.Mutex
	catch   exception.Matcher
	pending []func() error
	nested  *Finalizer
	index   int
}

// BindToReader moves the pending finalizations to the close action of a reader closer.
func (f *Finalizer) BindToReader(r io.ReadCloser, msg ...string) io.ReadCloser {
	f.lock.Lock()
	defer f.lock.Unlock()

	if len(f.pending) == 0 {
		addToCloser(r, nil, msg...)
	}
	n := &Finalizer{
		pending: f.pending,
	}
	f.pending = nil
	f.nested = nil
	f.index = 0
	return addToCloser(r, n, msg...)
}

// CatchException marks the finalizer to catch exceptions.
// This must be combined with error propagating defers.
func (f *Finalizer) CatchException(matchers ...exception.Matcher) *Finalizer {
	if len(matchers) > 0 {
		f.catch = exception.Or(matchers...)
	} else {
		f.catch = exception.All
	}
	return f
}

// Lock locks a given Locker and unlocks it again
// during finalization.
func (f *Finalizer) Lock(locker sync.Locker, msg ...string) *Finalizer {
	locker.Lock()
	return f.WithVoid(locker.Unlock, msg...)
}

// WithVoid registers a simple function to be
// called on finalization.
func (f *Finalizer) WithVoid(fi func(), msg ...string) *Finalizer {
	return f.With(CallingV(fi), msg...)
}

func (f *Finalizer) With(fi func() error, msg ...string) *Finalizer {
	if fi != nil {
		f.lock.Lock()
		defer f.lock.Unlock()

		if len(msg) > 0 {
			ofi := fi
			fi = func() error {
				err := ofi()
				if err == nil {
					return nil
				}
				return errors.Wrapf(err, "%s", strings.Join(msg, " "))
			}
		}
		f.pending = append(f.pending, fi)
	}
	return f
}

// Calling1 can be used with Finalizer.With, to call an error providing
// function with one argument.
func Calling1[T any](f func(arg T) error, arg T) func() error {
	return func() error {
		return f(arg)
	}
}

func Calling2[T, U any](f func(arg1 T, arg2 U) error, arg1 T, arg2 U) func() error {
	return func() error {
		return f(arg1, arg2)
	}
}

func Calling3[T, U, V any](f func(arg1 T, arg2 U, arg3 V) error, arg1 T, arg2 U, arg3 V) func() error {
	return func() error {
		return f(arg1, arg2, arg3)
	}
}

func CallingV(f func()) func() error {
	return func() error {
		f()
		return nil
	}
}

// Calling1V can be used with Finalizer.With, to call a void
// function with one argument.
func Calling1V[T any](f func(arg T), arg T) func() error {
	return func() error {
		f(arg)
		return nil
	}
}

func Calling2V[T, U any](f func(arg1 T, arg2 U), arg1 T, arg2 U) func() error {
	return func() error {
		f(arg1, arg2)
		return nil
	}
}

func Calling3V[T, U, V any](f func(arg1 T, arg2 U, arg3 V), arg1 T, arg2 U, arg3 V) func() error {
	return func() error {
		f(arg1, arg2, arg3)
		return nil
	}
}

// Close will finalize the given object by calling
// its Close function when the finalizer is finalized.
func (f *Finalizer) Close(c io.Closer, msg ...string) *Finalizer {
	if c != nil {
		f.With(c.Close, msg...)
	}
	return f
}

// Closef will finalize the given object by calling
// its Close function when the finalizer is finalized
// and annotates an error with the given formatted message.
func (f *Finalizer) Closef(c io.Closer, msg string, args ...interface{}) *Finalizer {
	if c != nil {
		f.With(c.Close, fmt.Sprintf(msg, args...))
	}
	return f
}

// ClosingWith can be used add a close request to
// finalizer in a chained call.
// Unfortunately it is not possible in Go
// to define parameterized methods, therefore
// we cannot directly add this function to the
// Finalizer type.
func ClosingWith[T io.Closer](f *Finalizer, o T) T {
	f.Close(o)
	return o
}

// Include includes the finalization of a given
// finalizer.
func (f *Finalizer) Include(fi *Finalizer) *Finalizer {
	if fi != nil {
		f.With(fi.Finalize)
	}
	return f
}

// New return a new finalizer included in the actual one.
func (f *Finalizer) New() *Finalizer {
	n := &Finalizer{}
	f.Include(n)
	return n
}

// Nested returns a linked finalizer usable in a nested block,
// which can be separately finalized. It is intended for sequential
// use, for example in a for loop. Successive calls
// will provide the same finalizer. The nested finalizer
// SHOULD be finalized at the end of its scope before
// it is requested, again, for the next nested usage.
func (f *Finalizer) Nested() *Finalizer {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.nested == nil || f.nested.Length() > 0 {
		f.nested = &Finalizer{}
	} else {
		f.pending = append(f.pending[:f.index], f.pending[f.index+1:]...)
	}
	f.index = len(f.pending)
	f.pending = append(f.pending, f.nested.Finalize)
	return f.nested
}

func (f *Finalizer) Length() int {
	f.lock.Lock()
	defer f.lock.Unlock()
	return len(f.pending)
}

// FinalizeWithErrorPropagation calls all finalizations in the reverse order of
// their registration and propagates a potential error to the given error
// variable incorporating an already existing error.
// This is especially intended to be used in a deferred mode to adapt
// the error code of a function to incorporate finalization errors.
func (f *Finalizer) FinalizeWithErrorPropagation(efferr *error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	errors.PropagateError(efferr, f.finalize)
	if f.catch == nil {
		return
	}
	if r := recover(); r != nil {
		if e := exception.Exception(r); e != nil && f.catch(e) {
			*efferr = errors.ErrList().Add(e).Add(*efferr).Result()
		} else {
			panic(r)
		}
	}
}

// FinalizeWithErrorPropagationf calls all finalizations in the reverse order of
// their registration and propagates a potential error to the given error
// variable incorporating an already existing error.
// This is especially intended to be used in a deferred mode to adapt
// the error code of a function to incorporate finalization errors.
// The final error will be wrapped by the given common context.
func (f *Finalizer) FinalizeWithErrorPropagationf(efferr *error, msg string, args ...interface{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	errors.PropagateErrorf(efferr, f.finalize, msg, args...)
	if f.catch == nil {
		return
	}
	if r := recover(); r != nil {
		if e := exception.Exception(r); e != nil && f.catch(e) {
			*efferr = errors.ErrList().Add(e).Add(*efferr).Result()
		} else {
			panic(r)
		}
	}
}

// Finalize calls all finalizations in the reverse order of
// their registration and incorporates catched exceptions.
func (f *Finalizer) Finalize() (err error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	err = f.finalize()
	if f.catch == nil {
		return err
	}
	if r := recover(); r != nil {
		if e := exception.Exception(r); e != nil && f.catch(e) {
			err = errors.ErrList().Add(e).Add(err).Result()
		} else {
			panic(r)
		}
	}
	return err
}

// ThrowFinalize executes the finalization and in case
// of an error it throws the error to be catched by an outer
// finalize or other error handling with the exception package.
// It is explicitly useful fo finalize nested finalizers in loops.
func (f *Finalizer) ThrowFinalize() {
	f.lock.Lock()
	defer f.lock.Unlock()

	err := f.finalize()
	if f.catch == nil {
		throwFinalizationError(err)
	}
	if r := recover(); r != nil {
		if e := exception.Exception(r); e != nil && f.catch(e) {
			err = errors.ErrList().Add(e).Add(err).Result()
		} else {
			panic(r)
		}
	}
	throwFinalizationError(err)
}

func (f *Finalizer) finalize() (err error) {
	list := errors.ErrList()
	l := len(f.pending)
	for i := range f.pending {
		list.Add(f.pending[l-i-1]())
	}
	f.pending = nil
	// just forget nested ones. They are finalized here, but are then invalid.
	// Adding entries after the parent has been finalized is not supported, because there is no valid determinable order.
	f.nested = nil
	return list.Result()
}

type FinalizationError struct {
	error
}

func (e FinalizationError) Unwrap() error {
	return e.error
}

// FinalizeException is an exception matcher for nested finalization exceptions.
func FinalizeException(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(FinalizationError) //nolint:errorlint // only for unwrapped error intended
	return ok
}

func throwFinalizationError(err error) {
	if err == nil {
		return
	}
	exception.Throw(FinalizationError{err})
}
