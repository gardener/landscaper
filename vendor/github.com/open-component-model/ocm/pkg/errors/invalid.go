// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

type InvalidError struct {
	errinfo
}

var formatInvalid = NewDefaultFormatter("is", "invalid", "for")

func ErrInvalid(spec ...string) error {
	return &InvalidError{newErrInfo(formatInvalid, spec...)}
}

func ErrInvalidWrap(err error, spec ...string) error {
	return &InvalidError{wrapErrInfo(err, formatInvalid, spec...)}
}

func IsErrInvalid(err error) bool {
	return IsA(err, &InvalidError{})
}

func IsErrInvalidKind(err error, kind string) bool {
	var uerr *InvalidError
	if err == nil || !As(err, &uerr) {
		return false
	}
	return uerr.kind == kind
}
