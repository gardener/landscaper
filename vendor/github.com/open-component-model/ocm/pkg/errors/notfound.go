// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

type NotFoundError struct {
	errinfo
}

var formatNotFound = NewDefaultFormatter("", "not found", "in")

func ErrNotFound(spec ...string) error {
	return &NotFoundError{newErrInfo(formatNotFound, spec...)}
}

func ErrNotFoundWrap(err error, spec ...string) error {
	return &NotFoundError{wrapErrInfo(err, formatNotFound, spec...)}
}

func IsErrNotFound(err error) bool {
	return IsA(err, &NotFoundError{})
}

func IsErrNotFoundKind(err error, kind string) bool {
	var uerr *NotFoundError
	if err == nil || !As(err, &uerr) {
		return false
	}
	return uerr.kind == kind
}
