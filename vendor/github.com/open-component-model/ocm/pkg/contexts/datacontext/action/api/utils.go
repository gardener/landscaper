// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/open-component-model/ocm/pkg/runtime"
)

type _Object = runtime.ObjectVersionedTypedObject

type actionType struct {
	_Object
	spectype ActionSpecType
	restype  ActionResultType
}

var _ ActionType = (*actionType)(nil)

func NewActionType[IS ActionSpec, IR ActionResult](kind, version string) ActionType {
	return NewActionTypeByConverter[IS, IS, IR, IR](kind, version, runtime.IdentityConverter[IS]{}, runtime.IdentityConverter[IR]{})
}

func NewActionTypeByConverter[IS ActionSpec, VS runtime.TypedObject, IR ActionResult, VR runtime.TypedObject](kind, version string, specconv runtime.Converter[IS, VS], resconv runtime.Converter[IR, VR]) ActionType {
	name := runtime.TypeName(kind, version)
	st := runtime.NewVersionedTypedObjectTypeByConverter[ActionSpec, IS, VS](name, specconv)
	rt := runtime.NewVersionedTypedObjectTypeByConverter[ActionResult, IR, VR](name, resconv)
	return &actionType{
		_Object:  runtime.NewVersionedTypedObject(kind, version),
		spectype: st,
		restype:  rt,
	}
}

func (a *actionType) SpecificationType() ActionSpecType {
	return a.spectype
}

func (a *actionType) ResultType() ActionResultType {
	return a.restype
}
