// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package descriptor

import (
	"sort"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/utils/registry"
	"github.com/open-component-model/ocm/pkg/generics"
)

type Named interface {
	GetName() string
}

type StringName string

func (e StringName) GetName() string {
	return string(e)
}

type Element[K registry.Key[K]] interface {
	Named
	GetConstraints() []K
}

type List[T Named] []T

func (l List[T]) Get(name string) *T {
	for _, m := range l {
		if m.GetName() == name {
			return &m
		}
	}
	return nil
}

func (l List[T]) GetNames() []string {
	var n []string
	for _, e := range l {
		n = append(n, e.GetName())
	}
	sort.Strings(n)
	return n
}

func (l List[T]) MergeWith(o List[T]) List[T] {
	var list []T
next:
	for _, e := range o {
		for _, f := range l {
			if e.GetName() == f.GetName() {
				continue next
			}
		}
		list = append(list, e)
	}
	return generics.AppendedSlice(l, list...)
}
