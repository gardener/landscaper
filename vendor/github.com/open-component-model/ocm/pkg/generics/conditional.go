// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

func Conditional[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
