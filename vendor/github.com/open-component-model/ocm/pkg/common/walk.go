// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"github.com/open-component-model/ocm/pkg/utils"
)

type NameVersionInfo[T any] map[NameVersion]T

func (s NameVersionInfo[T]) Add(nv NameVersion, data ...T) bool {
	if _, ok := s[nv]; !ok {
		s[nv] = utils.Optional(data...)
		return true
	}
	return false
}

func (s NameVersionInfo[T]) Contains(nv NameVersion) bool {
	_, ok := s[nv]
	return ok
}

type WalkingState[T any] struct {
	Closure NameVersionInfo[T]
	History History
}

func NewWalkingState[T any]() WalkingState[T] {
	return WalkingState[T]{Closure: NameVersionInfo[T]{}}
}

func (s *WalkingState[T]) Add(kind string, nv NameVersion) (bool, error) {
	if err := s.History.Add(kind, nv); err != nil {
		return false, err
	}
	return s.Closure.Add(nv), nil
}

func (s *WalkingState[T]) Contains(nv NameVersion) bool {
	_, ok := s.Closure[nv]
	return ok
}

func (s *WalkingState[T]) Get(nv NameVersion) T {
	return s.Closure[nv]
}