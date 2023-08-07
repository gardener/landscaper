// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

// Ptr returns a pointer to the given object.
func Ptr[T any](value T) *T {
	return &value
}
