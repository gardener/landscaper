// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

import (
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type Set[K comparable] map[K]struct{}

func NewSet[K comparable](keys ...K) Set[K] {
	return Set[K]{}.Add(keys...)
}

func (s Set[K]) Add(keys ...K) Set[K] {
	for _, k := range keys {
		s[k] = struct{}{}
	}
	return s
}

func (s Set[K]) Delete(keys ...K) Set[K] {
	for _, k := range keys {
		delete(s, k)
	}
	return s
}

func (s Set[K]) Contains(keys ...K) bool {
	for _, k := range keys {
		if _, ok := s[k]; !ok {
			return false
		}
	}
	return true
}

func (s Set[K]) AsArray() []K {
	keys := []K{}
	for k := range s {
		keys = append(keys, k)
	}
	return keys
}

func KeySet[K comparable, V any](m map[K]V) Set[K] {
	s := Set[K]{}
	for k := range m {
		s.Add(k)
	}
	return s
}

type Comparable[K any] interface {
	comparable
	Compare(o K) int
}

func KeyList[K Comparable[K], V any](m map[K]V) []K {
	s := maps.Keys(m)
	slices.SortFunc(s, func(a, b K) bool { return a.Compare(b) < 0 })
	return s
}
