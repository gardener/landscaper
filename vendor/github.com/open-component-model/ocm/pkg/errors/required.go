// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

type RequiredError struct {
	errinfo
}

var formatRequired = NewDefaultFormatter("", "required", "for")

func ErrRequired(spec ...string) error {
	return &RequiredError{newErrInfo(formatRequired, spec...)}
}

func ErrRequiredWrap(err error, spec ...string) error {
	return &RequiredError{wrapErrInfo(err, formatRequired, spec...)}
}

func IsErrNRequired(err error) bool {
	return IsA(err, &RequiredError{})
}

func IsErrRequiredKind(err error, kind string) bool {
	var uerr *RequiredError
	if err == nil || !As(err, &uerr) {
		return false
	}
	return uerr.kind == kind
}

func IsErrRequiredElem(err error, kind, elem string) bool {
	var uerr *RequiredError
	if err == nil || !As(err, &uerr) {
		return false
	}
	return uerr.kind == kind && uerr.elem != nil && *uerr.elem == elem
}
