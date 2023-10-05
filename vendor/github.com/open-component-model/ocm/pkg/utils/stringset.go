// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

type StringSet map[string]struct{}

func (s StringSet) Add(a string) bool {
	if _, ok := s[a]; ok {
		return false
	}
	s[a] = struct{}{}
	return true
}

func (s StringSet) Remove(a string) bool {
	if _, ok := s[a]; !ok {
		return false
	}
	delete(s, a)
	return true
}

func (s StringSet) Contains(a string) bool {
	_, ok := s[a]
	return ok
}
