// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"reflect"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/equivalent"
)

func (cd *ComponentDescriptor) Equal(obj interface{}) bool {
	if o, ok := obj.(*ComponentDescriptor); ok {
		if !cd.ObjectMeta.Equal(&o.ObjectMeta) {
			return false
		}
		if !reflect.DeepEqual(cd.Sources, o.Sources) {
			return false
		}
		if !reflect.DeepEqual(cd.Resources, o.Resources) {
			return false
		}
		if !reflect.DeepEqual(cd.References, o.References) {
			return false
		}
		if !reflect.DeepEqual(cd.Signatures, o.Signatures) {
			return false
		}
		if !reflect.DeepEqual(cd.NestedDigests, o.NestedDigests) {
			return false
		}
		return true
	}
	return false
}

func (cd *ComponentDescriptor) Equivalent(o *ComponentDescriptor) equivalent.EqualState {
	return equivalent.StateEquivalent().Apply(
		cd.ObjectMeta.Equivalent(o.ObjectMeta),
		cd.Resources.Equivalent(o.Resources),
		cd.Sources.Equivalent(o.Sources),
		cd.References.Equivalent(o.References),
		cd.Signatures.Equivalent(o.Signatures),
	)
}

func EquivalentElems(a ElementAccessor, b ElementAccessor) equivalent.EqualState {
	state := equivalent.StateEquivalent()

	// Equivaluent of elements handles nil to provide state accoding to it
	// relevance for the signature.
	for i := 0; i < a.Len(); i++ {
		ea := a.Get(i)

		ib := GetIndexByIdentity(b, ea.GetMeta().GetIdentity(a))
		if ib != i {
			state = state.NotLocalHashEqual()
		}

		var eb ElementMetaAccessor
		if ib >= 0 {
			eb = b.Get(ib)
		}
		state = state.Apply(ea.Equivalent(eb))
	}
	for i := 0; i < b.Len(); i++ {
		eb := b.Get(i)
		if ea := GetByIdentity(a, eb.GetMeta().GetIdentity(b)); ea == nil {
			state = state.Apply(eb.Equivalent(ea))
		}
	}
	return state
}
