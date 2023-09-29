// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

type ReadOnlyError struct {
	errinfo
}

var formatReadOnly = NewDefaultFormatter("is", "readonly", "in")

func ErrReadOnly(spec ...string) error {
	return &ReadOnlyError{newErrInfo(formatReadOnly, spec...)}
}

func IsErrReadOnly(err error) bool {
	return IsA(err, &ReadOnlyError{})
}

func IsErrReadOnlyKind(err error, kind string) bool {
	var uerr *ReadOnlyError
	if err == nil || !As(err, &uerr) {
		return false
	}
	return uerr.kind == kind
}
