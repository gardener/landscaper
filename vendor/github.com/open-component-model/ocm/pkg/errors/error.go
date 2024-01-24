// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// //nolint: errorlint // this is the new As method, also handling error lists
package errors

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	New    = errors.New
	Unwrap = errors.Unwrap
)

func Newf(msg string, args ...interface{}) error {
	return New(fmt.Sprintf(msg, args...))
}

func As(err error, target any) bool {
	if errors.As(err, target) {
		return true
	}

	for err != nil {
		if list, ok := err.(*ErrorList); ok {
			for _, n := range list.errors {
				if As(n, target) {
					return true
				}
			}
		}
		err = Unwrap(err)
	}
	return false
}

// Is checks for a concrete error object
// along the error chain.
func Is(err error, target error) bool {
	if target == nil || err == nil {
		return err == target
	}

	if errors.Is(err, target) {
		return true
	}

	for err != nil {
		if list, ok := err.(*ErrorList); ok {
			for _, n := range list.errors {
				if Is(n, target) {
					return true
				}
			}
		}
		err = Unwrap(err)
	}
	return false
}

// IsA checks for an error of a dedicated type
// along the error chain.
func IsA(err error, target error) bool {
	if target == nil {
		return err == target
	}
	if err == nil {
		return false
	}
	typ := reflect.TypeOf(target)

	for err != nil {
		if reflect.TypeOf(err).AssignableTo(typ) {
			return true
		}
		if list, ok := err.(*ErrorList); ok {
			for _, n := range list.errors {
				if IsA(n, target) {
					return true
				}
			}
		}
		err = Unwrap(err)
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////

type wrappedError struct {
	wrapped error
	msg     string
}

// NewEf provides an arror with an optional cause.
func NewEf(cause error, msg string, args ...interface{}) error {
	if cause == nil {
		return Newf(msg, args...)
	}
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &wrappedError{
		wrapped: cause,
		msg:     msg,
	}
}

func Wrapf(err error, msg string, args ...interface{}) error {
	if err == nil || msg == "" {
		return err
	}
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &wrappedError{
		wrapped: err,
		msg:     msg,
	}
}

func Wrap(err error, args ...interface{}) error {
	if err == nil || len(args) == 0 {
		return err
	}
	msg := fmt.Sprint(args...)
	return &wrappedError{
		wrapped: err,
		msg:     msg,
	}
}

func (e *wrappedError) Error() string {
	return fmt.Sprintf("%s: %s", e.msg, e.wrapped)
}

func (e *wrappedError) Unwrap() error {
	return e.wrapped
}

// var errorType = reflect.TypeOf((*error)(nil)).Elem()

////////////////////////////////////////////////////////////////////////////////

type errinfo struct {
	wrapped error
	format  ErrorFormatter
	kind    string
	elem    *string
	ctxkind string
	ctx     string
}

func wrapErrInfo(err error, fmt ErrorFormatter, spec ...string) errinfo {
	e := newErrInfo(fmt, spec...)
	e.wrapped = err
	return e
}

func newErrInfo(fmt ErrorFormatter, spec ...string) errinfo {
	e := errinfo{
		format: fmt,
	}

	if len(spec) > 3 {
		e.kind = spec[0]
		e.elem = &spec[1]
		e.ctxkind = spec[2]
		e.ctx = spec[3]
		return e
	}
	if len(spec) > 2 {
		e.kind = spec[0]
		e.elem = &spec[1]
		e.ctx = spec[2]
		return e
	}
	if len(spec) > 1 {
		e.kind = spec[0]
		e.elem = &spec[1]
		return e
	}

	if len(spec) > 0 {
		e.elem = &spec[0]
	}
	return e
}

func (e *errinfo) Is(o error) bool {
	if oe, ok := o.(interface{ formatMessage() string }); ok {
		return oe.formatMessage() == e.formatMessage()
	}
	return false
}

func (e *errinfo) formatMessage() string {
	return e.format.Format(e.kind, e.elem, e.ctxkind, e.ctx)
}

func (e *errinfo) Error() string {
	msg := e.formatMessage()
	if e.wrapped != nil {
		return msg + ": " + e.wrapped.Error()
	}
	return msg
}

func (e *errinfo) Unwrap() error {
	return e.wrapped
}

func (e *errinfo) Elem() *string {
	return e.elem
}

func (e *errinfo) Kind() string {
	return e.kind
}

func (e *errinfo) CtxKind() string {
	return e.ctxkind
}

func (e *errinfo) Ctx() string {
	return e.ctx
}

type Kinded interface {
	Kind() string
	SetKind(string)
}

func (e *errinfo) SetKind(kind string) {
	e.kind = kind
}
