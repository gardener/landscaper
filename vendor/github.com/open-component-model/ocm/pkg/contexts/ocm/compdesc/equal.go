// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"reflect"
)

func (cd *ComponentDescriptor) Equal(o *ComponentDescriptor) bool {
	o = o.Copy()
	o.Metadata.ConfiguredVersion = cd.Metadata.ConfiguredVersion
	o.RepositoryContexts = cd.RepositoryContexts

	return reflect.DeepEqual(cd, o)
}

func (cd *ComponentDescriptor) Equivalent(o *ComponentDescriptor) bool {
	if !reflect.DeepEqual(&cd.ObjectMeta, &o.ObjectMeta) {
		return false
	}

	if !equivalentElems(cd.Resources, o.Resources) {
		return false
	}
	if !equivalentElems(cd.Sources, o.Sources) {
		return false
	}
	if !equivalentElems(cd.References, o.References) {
		return false
	}
	return true
}

func equivalentElems(a ElementAccessor, b ElementAccessor) bool {
	if a.Len() != b.Len() {
		return false
	}

	for i := 0; i < a.Len(); i++ {
		if !a.Get(i).IsEquivalent(b.Get(i)) {
			return false
		}
	}
	return true
}
