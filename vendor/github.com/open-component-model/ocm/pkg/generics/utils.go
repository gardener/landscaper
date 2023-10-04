// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

func Nil[T any]() T {
	var _nil T
	return _nil
}

func Pointer[T any](t T) *T {
	return &t
}

func Value[T any](t *T) T {
	if t != nil {
		return *t
	}
	var v T
	return v
}
