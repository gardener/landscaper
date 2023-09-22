// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

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
