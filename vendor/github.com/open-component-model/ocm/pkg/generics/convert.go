// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

// ConvertSlice Converts the element typ of a slice.
// hereby, Type S must be a subtype of type T.
func ConvertSlice[S, T any](in ...S) []T {
	if in == nil {
		return nil
	}
	r := make([]T, len(in))
	for i := range in {
		var s any = in[i]
		r[i] = As[T](s)
	}
	return r
}
