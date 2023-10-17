// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package clitypes

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type additionalCLITypeInfo interface {
	ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler
	Description() string
	Format() string
}

type CLITypedObjectTypeObject[E runtime.VersionedTypedObject] struct {
	runtime.VersionedTypedObjectType[E]
	description string
	format      string
	handler     flagsets.ConfigOptionTypeSetHandler
	validator   func(E) error
}

var _ additionalCLITypeInfo = (*CLITypedObjectTypeObject[runtime.VersionedTypedObject])(nil)

func NewCLITypedObjectTypeObject[E runtime.VersionedTypedObject](vt runtime.VersionedTypedObjectType[E], opts ...CLITypeOption) *CLITypedObjectTypeObject[E] {
	t := CLIObjectTypeTarget[E]{&CLITypedObjectTypeObject[E]{
		VersionedTypedObjectType: vt,
	}}
	for _, o := range opts {
		o.ApplyToCLIOptionTarget(t)
	}
	return t.target
}

func (t *CLITypedObjectTypeObject[E]) ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler {
	return t.handler
}

func (t *CLITypedObjectTypeObject[E]) Description() string {
	return t.description
}

func (t *CLITypedObjectTypeObject[E]) Format() string {
	return t.format
}

func (t *CLITypedObjectTypeObject[E]) Validate(e E) error {
	if t.validator == nil {
		return nil
	}
	return t.validator(e)
}

////////////////////////////////////////////////////////////////////////////////

// CLIObjectTypeTarget is used as target for option functions, it provides
// setters for fields, which should nor be modifiable for a final type object.
type CLIObjectTypeTarget[E runtime.VersionedTypedObject] struct {
	target *CLITypedObjectTypeObject[E]
}

func (t CLIObjectTypeTarget[E]) SetDescription(value string) {
	t.target.description = value
}

func (t CLIObjectTypeTarget[E]) SetFormat(value string) {
	t.target.format = value
}

func (t CLIObjectTypeTarget[E]) SetConfigHandler(value flagsets.ConfigOptionTypeSetHandler) {
	t.target.handler = value
}
