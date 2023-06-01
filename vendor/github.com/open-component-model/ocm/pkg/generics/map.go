// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

func MapKeys[K comparable, V any](m map[K]V) Set[K] {
	s := Set[K]{}
	for k := range m {
		s.Add(k)
	}
	return s
}

func MapKeyArray[K comparable, V any](m map[K]V) []K {
	a := make([]K, len(m))
	i := 0
	for k := range m {
		a[i] = k
		i++
	}
	return a
}

func MapValues[K comparable, V any](m map[K]V) []V {
	a := make([]V, len(m))
	i := 0
	for _, v := range m {
		a[i] = v
		i++
	}
	return a
}
