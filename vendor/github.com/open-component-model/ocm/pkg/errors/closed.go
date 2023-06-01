// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

type ClosedError struct {
	errinfo
}

var formatClosed = NewDefaultFormatter("is", "closed", "for")

func ErrClosed(spec ...string) error {
	return &ClosedError{newErrInfo(formatClosed, spec...)}
}

func IsErrClosed(err error) bool {
	return IsA(err, &ClosedError{})
}

func IsErrClosedKind(err error, kind string) bool {
	var uerr *ClosedError
	if err == nil || !As(err, &uerr) {
		return false
	}
	return uerr.kind == kind
}
