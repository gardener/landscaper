// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package exception provides a simple exception mechanism
// to reduce boilerplate for trivial error forwarding in
// a function.
// Example:
//
//	func f0() error {
//	  return nil
//	}
//
//	func f1() (int, error) {
//	  return 1, nil
//	}
//
//	func MyFunc() (err error) {
//	  defer PropagateException(&err)
//
//	  Mustf(f0(),"f0 failed")
//	  i:=Must1f(R1(f1()), "f1 failed")
//	  fmt.Printf("got %d\n", i)
//	  return nil
//	}
package exception

import (
	"github.com/open-component-model/ocm/pkg/errors"
)

type Matcher func(err error) bool

// PropagateException catches an exception provided by variations of
// MustX functions and forwards the error to its argument.
// It must be called by defer.
// Cannot reuse PropagateExceptionf, because recover MUST be called
// at TOP level defer function to recover the panic.
func PropagateException(errp *error, matchers ...Matcher) {
	if r := recover(); r != nil {
		*errp = FilterException(r, matchers...)
	}
}

// FilterException can be used in a own defer function
// to handle exceptions.
// In Go it is not possible to provide the complete catch in
// a function, because recover works on top-level functions, only.
// Therefore. the recover call has to be placed directly in the
// deferred wrapper function, which can then use this function to catch
// an exception and convert it to an error code.
func FilterException(r interface{}, matchers ...Matcher) error {
	if e, ok := r.(*exception); ok && match(e.err, matchers...) {
		return e.err
	} else {
		panic(r)
	}
}

// CatchError calls the given function with the error of
// a catched exception, if it is deferred.
func CatchError(f func(err error), matchers ...Matcher) {
	if r := recover(); r != nil {
		err := FilterException(r, matchers...)
		if err != nil {
			f(err)
		}
	}
}

func match(err error, matchers ...Matcher) bool {
	if len(matchers) == 0 {
		return true
	}
	for _, m := range matchers {
		if m(err) {
			return true
		}
	}
	return false
}

var All = func(_ error) bool {
	return true
}

var None = func(_ error) bool {
	return false
}

func ByPrototypes(protos ...error) Matcher {
	return func(err error) bool {
		if len(protos) == 0 {
			return true
		}
		for _, p := range protos {
			if errors.IsA(err, p) {
				return true
			}
		}
		return false
	}
}

func Or(matchers ...Matcher) Matcher {
	return func(err error) bool {
		for _, m := range matchers {
			if m(err) {
				return true
			}
		}
		return false
	}
}

func And(matchers ...Matcher) Matcher {
	return func(err error) bool {
		for _, m := range matchers {
			if !m(err) {
				return false
			}
		}
		return true
	}
}

// Exception provie the error object from an exception object.
func Exception(r interface{}) error {
	if e, ok := r.(*exception); ok {
		return e.err
	}
	return nil
}

func Catch(funcs ...func()) (err error) {
	PropagateException(&err)
	for _, f := range funcs {
		f()
	}
	return nil
}

// PropagateExceptionf catches an exception provided by variations of
// MustX functions and forwards the error to its argument.
// It must be called by defer.
// If an error context is given (msg!=""), it is added to the propagated error.
func PropagateExceptionf(errp *error, msg string, args ...interface{}) {
	if r := recover(); r != nil {
		if e, ok := r.(*exception); ok {
			if msg != "" {
				*errp = errors.Wrapf(e.err, msg, args...)
			} else {
				*errp = e.err
			}
		} else {
			panic(r)
		}
	}
}

func PropagateMatchedExceptionf(errp *error, m Matcher, msg string, args ...interface{}) {
	if r := recover(); r != nil {
		if e, ok := r.(*exception); ok && (m == nil || m(e.err)) {
			if msg != "" {
				*errp = errors.Wrapf(e.err, msg, args...)
			} else {
				*errp = e.err
			}
		} else {
			panic(r)
		}
	}
}

// ForwardExceptionf add an error context to a forwarded exception.
// Usage: defer ForwardExceptionf("error context").
func ForwardExceptionf(msg string, args ...interface{}) {
	if r := recover(); r != nil {
		if e, ok := r.(*exception); ok {
			if msg != "" {
				e.err = errors.Wrapf(e.err, msg, args...)
			}
			panic(e)
		} else {
			panic(r)
		}
	}
}

type exception struct {
	err error
}

func (e *exception) Unwrap() error {
	return e.err
}

func (e *exception) Error() string {
	return e.err.Error()
}

// Throw throws an exception if err !=nil.
func Throw(err error) {
	if err == nil {
		return
	}
	panic(&exception{err})
}

// Throwf throws an exception if err !=nil.
// A given error context is given, it wraps the
// error by the context.
func Throwf(err error, msg string, args ...interface{}) {
	if err == nil {
		return
	}
	if msg == "" {
		panic(&exception{err})
	}
	panic(&exception{errors.Wrapf(err, msg, args...)})
}

// Must converts an error (e.g. provided by a nested function call) into an
// exception. It provides no regular result.
func Must(err error) {
	Throw(err)
}

// Must converts an error (e.g. provided by a nested function call) into an
// exception. It provides no regular result.
// The given error context is used to wrap the error.
func Mustf(err error, msg string, args ...interface{}) {
	Throwf(err, msg, args...)
}

////////////////////////////////////////////////////////////////////////////////

type result1[A any] struct {
	r1  A
	err error
}

// R1 bundles the result of an error function with one additional
// argument to be passed to Must1f.
func R1[A any](r1 A, err error) result1[A] { return result1[A]{r1, err} }

// Must1 converts an error into an exception. It provides one regular
// result.
//
// Usage: Must1(ErrorFunctionWithOneRegularResult()).
func Must1[A any](r1 A, err error) A {
	Throw(err)
	return r1
}

// Must1f converts an error into an exception. It provides one regular
// result, which has to be provided by method R1.
// Optionally an error context can be given.
//
// Usage: Must1f(R1(ErrorFunctionWithOneRegularResult()), "context").
//
// The intermediate function R1 is required , because GO does not
// allow to compose arguments provided by a function with multiple
// return values with additional arguments.
func Must1f[A any](r result1[A], msg string, args ...interface{}) A {
	Throwf(r.err, msg, args...)
	return r.r1
}

////////////////////////////////////////////////////////////////////////////////

type result2[A, B any] struct {
	r1  A
	r2  B
	err error
}

// R2 bundles the result of an error function with two additional
// arguments to be passed to Must1.
func R2[A, B any](r1 A, r2 B, err error) result2[A, B] { return result2[A, B]{r1, r2, err} }

// Must2 converts an error into an exception. It provides two regular
// results.
func Must2[A, B any](r1 A, r2 B, err error) (A, B) {
	Throw(err)
	return r1, r2
}

// Must2f like Must1f, but for two regular
// results, which has to be provided by method R2.
// The error is wrapped by the given error context.
func Must2f[A, B any](r result2[A, B], msg string, args ...interface{}) (A, B) {
	Throwf(r.err, msg, args...)
	return r.r1, r.r2
}

////////////////////////////////////////////////////////////////////////////////

type result3[A, B, C any] struct {
	r1  A
	r2  B
	r3  C
	err error
}

// R3 bundles the result of an error function with three additional
// arguments to be passed to Must3.
func R3[A, B, C any](r1 A, r2 B, r3 C, err error) result3[A, B, C] {
	return result3[A, B, C]{r1, r2, r3, err}
}

// Must3 converts an error into an exception. It provides three regular
// results.
func Must3[A, B, C any](r1 A, r2 B, r3 C, err error) (A, B, C) {
	Throw(err)
	return r1, r2, r3
}

// Must3f like Must1f, but for three regular
// results, which has to be provided by method R3.
func Must3f[A, B, C any](r result3[A, B, C], msg string, args ...interface{}) (A, B, C) {
	Throwf(r.err, msg, args...)
	return r.r1, r.r2, r.r3
}

////////////////////////////////////////////////////////////////////////////////

type result4[A, B, C, D any] struct {
	r1  A
	r2  B
	r3  C
	r4  D
	err error
}

// R4 bundles the result of an error function with four additional
// arguments to be passed to Must4.
func R4[A, B, C, D any](r1 A, r2 B, r3 C, r4 D, err error) result4[A, B, C, D] {
	return result4[A, B, C, D]{r1, r2, r3, r4, err}
}

// Must4 converts an error into an exception. It provides four regular
// results.
func Must4[A, B, C, D any](r1 A, r2 B, r3 C, r4 D, err error) (A, B, C, D) {
	Throw(err)
	return r1, r2, r3, r4
}

// Must4f like Must1f, but for four regular
// results, which has to be provided by method R4.
func Must4f[A, B, C, D any](r result4[A, B, C, D], msg string, args ...interface{}) (A, B, C, D) {
	Throwf(r.err, msg, args...)
	return r.r1, r.r2, r.r3, r.r4
}
