// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

// ConvertSlice Converts the element typ of a slice.
// hereby, Type S must be a subtype of type T.
func ConvertSlice[S, T any](in ...S) []T {
	return ConvertSliceTo[T, S](in)
}

// ConvertSliceTo Converts the element typ of a slice.
// hereby, Type S must be a subtype of type T.
// typically, S can be omitted when used, because it can be
// derived from the argument.
func ConvertSliceTo[T, S any](in []S) []T {
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

// ConvertSliceWith converts the element type of a slice
// using a converter function.
// Unfortunately this cannot be expressed in a type-safe way in Go.
// I MUST follow the type constraint I super S, which cannot be expressed in Go.
func ConvertSliceWith[S, T, I any](c func(I) T, in []S) []T {
	if in == nil {
		return nil
	}
	r := make([]T, len(in))
	for i := range in {
		var s any = in[i]
		r[i] = c(As[I](s))
	}
	return r
}
