// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

// Ptr returns a pointer to the given object.
func Ptr[T any](value T) *T {
	return &value
}

// MergeMaps merges multiple maps into a single one.
// In case of duplicate keys, the value from the last argument which had the key overwrites the previous ones.
// The returned map is a new one, the given maps are not modified.
func MergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	res := map[K]V{}

	for _, m := range maps {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}
