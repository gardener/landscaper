// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

type Unwrappable interface {
	Unwrap() interface{}
}

func Unwrap(o interface{}) interface{} {
	if o != nil {
		if u, ok := o.(Unwrappable); ok {
			return u.Unwrap()
		}
	}
	return nil
}

func UnwrappingCast[I interface{}](o interface{}) I {
	var _nil I

	for o != nil {
		if i, ok := o.(I); ok {
			return i
		}
		if i := Unwrap(o); i != o {
			o = i
		} else {
			o = nil
		}
	}
	return _nil
}

func UnwrappingCall[R any, I any](o interface{}, f func(I) R) R {
	var _nil R

	for o != nil {
		if i, ok := o.(I); ok {
			return f(i)
		}
		if i := Unwrap(o); i != o {
			o = i
		} else {
			o = nil
		}
	}
	return _nil
}
