// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

type AlreadyExistsError struct {
	errinfo
}

var formatAlreadyExists = NewDefaultFormatter("", "already exists", "in")

func ErrAlreadyExists(spec ...string) error {
	return &AlreadyExistsError{newErrInfo(formatAlreadyExists, spec...)}
}

func ErrAlreadyExistsWrap(err error, spec ...string) error {
	return &AlreadyExistsError{wrapErrInfo(err, formatAlreadyExists, spec...)}
}

func IsErrAlreadyExists(err error) bool {
	return IsA(err, &AlreadyExistsError{})
}

func IsErrAlreadyExistsKind(err error, kind string) bool {
	var uerr *NotFoundError
	if err == nil || !As(err, &uerr) {
		return false
	}
	return uerr.kind == kind
}
