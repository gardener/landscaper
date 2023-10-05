// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"fmt"
	"io"
)

// Join combines any number of errors to a single error.
// If no or only nil errors are given nil is returned.
// If only one effective error is provided, this error is returned.
func Join(errs ...error) error {
	return (&ErrorList{}).Add(errs...).Result()
}

// ErrorList is an error type with erros in it.
type ErrorList struct { //nolint: errname // Intentional naming.
	msg    string
	errors []error
}

func (l *ErrorList) Error() string {
	msg := ""
	if l.msg != "" {
		msg = fmt.Sprintf("%s: ", l.msg)
	}

	if len(l.errors) == 1 {
		return fmt.Sprintf("%s%s", msg, l.errors[0].Error())
	}
	sep := "{"
	for _, e := range l.errors {
		if e != nil {
			msg = fmt.Sprintf("%s%s%s", msg, sep, e)
			sep = ", "
		}
	}
	return msg + "}"
}

func (l *ErrorList) Add(errs ...error) *ErrorList {
	for _, e := range errs {
		if e != nil {
			l.errors = append(l.errors, e)
		}
	}
	return l
}

func (l *ErrorList) Addf(writer io.Writer, err error, msg string, args ...interface{}) error {
	if err != nil {
		if msg != "" {
			err = Wrapf(err, msg, args...)
		}
		l.errors = append(l.errors, err)
		if writer != nil {
			fmt.Fprintf(writer, "Error: %s\n", err)
		}
	}
	return err
}

func (l *ErrorList) Len() int {
	return len(l.errors)
}

func (l *ErrorList) Entries() []error {
	return l.errors
}

func (l *ErrorList) Result() error {
	if l == nil || len(l.errors) == 0 {
		return nil
	}
	if l.msg == "" && len(l.errors) == 1 {
		return l.errors[0]
	}
	return l
}

func (l *ErrorList) Clear() {
	l.errors = nil
}

func ErrListf(msg string, args ...interface{}) *ErrorList {
	return &ErrorList{
		msg: fmt.Sprintf(msg, args...),
	}
}

func ErrList(args ...interface{}) *ErrorList {
	return &ErrorList{
		msg: fmt.Sprint(args...),
	}
}
