// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"io"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
)

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
	pending []func() error
	nested  *Finalizer
}

// Lock locks a given Locker and unlocks it again
// during finalization.
func (f *Finalizer) Lock(locker sync.Locker) *Finalizer {
	locker.Lock()
	return f.WithVoid(locker.Unlock)
}

// WithVoid registers a simple function to be
// called on finalization.
func (f *Finalizer) WithVoid(fi func()) *Finalizer {
	return f.With(func() error { fi(); return nil })
}

func (f *Finalizer) With(fi func() error) *Finalizer {
	if fi != nil {
		f.lock.Lock()
		defer f.lock.Unlock()

		f.pending = append(f.pending, fi)
	}
	return f
}

// Close will finalize the given object by calling
// its Close function when the finalizer is finalized.
func (f *Finalizer) Close(c io.Closer) *Finalizer {
	if c != nil {
		f.With(c.Close)
	}
	return f
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

	if f.nested == nil {
		f.nested = &Finalizer{}
		f.pending = append(f.pending, f.nested.Finalize)
	}
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
	errors.PropagateError(efferr, f.Finalize)
}

// FinalizeWithErrorPropagationf calls all finalizations in the reverse order of
// their registration and propagates a potential error to the given error
// variable incorporating an already existing error.
// This is especially intended to be used in a deferred mode to adapt
// the error code of a function to incorporate finalization errors.
// The final error will be wrapped by the given common context.
func (f *Finalizer) FinalizeWithErrorPropagationf(efferr *error, msg string, args ...interface{}) {
	errors.PropagateErrorf(efferr, f.Finalize, msg, args...)
}

// Finalize calls all finalizations in the reverse order of
// their registration.
func (f *Finalizer) Finalize() error {
	f.lock.Lock()
	defer f.lock.Unlock()

	list := errors.ErrList()
	l := len(f.pending)
	for i := range f.pending {
		list.Add(f.pending[l-i-1]())
	}
	f.pending = nil
	return list.Result()
}
