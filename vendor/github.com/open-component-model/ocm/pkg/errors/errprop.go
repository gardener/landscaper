// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

type ErrorFunction func() error

// PropagateError propagates a deferred error to the named return value
// whose address has to be passed as argument.
func PropagateError(errp *error, f ErrorFunction) {
	PropagateErrorf(errp, f, "")
}

// PropagateErrorf propagates an optional deferred error to the named return value
// whose address has to be passed as argument.
// All errors, including the original one, are wrapped by the given context.
func PropagateErrorf(errp *error, f ErrorFunction, msg string, args ...interface{}) {
	if f == nil {
		*errp = ErrListf(msg, args...).Add(*errp).Result()
	} else {
		*errp = ErrListf(msg, args...).Add(*errp, f()).Result()
	}
}
