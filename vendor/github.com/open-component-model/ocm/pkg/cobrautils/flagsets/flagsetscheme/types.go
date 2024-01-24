// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsetscheme

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/optionutils"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/runtime/descriptivetype"
)

type additionalTypeInfo interface {
	ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler
	Description() string
	Format() string
}

type TypedObjectTypeObject[E runtime.VersionedTypedObject] struct {
	*descriptivetype.TypedObjectTypeObject[E]
	handler   flagsets.ConfigOptionTypeSetHandler
	validator func(E) error
}

var _ additionalTypeInfo = (*TypedObjectTypeObject[runtime.VersionedTypedObject])(nil)

func NewTypedObjectTypeObject[E runtime.VersionedTypedObject](vt runtime.VersionedTypedObjectType[E], opts ...TypeOption) *TypedObjectTypeObject[E] {
	t := NewTypeObjectTarget[E](&TypedObjectTypeObject[E]{
		TypedObjectTypeObject: descriptivetype.NewTypedObjectTypeObject[E](vt),
	})
	optionutils.ApplyOptions[OptionTarget](t, opts...)
	return t.target
}

func (t *TypedObjectTypeObject[E]) ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler {
	return t.handler
}

func (t *TypedObjectTypeObject[E]) Validate(e E) error {
	if t.validator == nil {
		return nil
	}
	return t.validator(e)
}

////////////////////////////////////////////////////////////////////////////////

// TypeObjectTarget is used as target for option functions, it provides
// setters for fields, which should nor be modifiable for a final type object.
type TypeObjectTarget[E runtime.VersionedTypedObject] struct {
	*descriptivetype.TypeObjectTarget[E]
	target *TypedObjectTypeObject[E]
}

func NewTypeObjectTarget[E runtime.VersionedTypedObject](target *TypedObjectTypeObject[E]) *TypeObjectTarget[E] {
	return &TypeObjectTarget[E]{
		target:           target,
		TypeObjectTarget: descriptivetype.NewTypeObjectTarget[E](target.TypedObjectTypeObject),
	}
}

func (t TypeObjectTarget[E]) SetConfigHandler(value flagsets.ConfigOptionTypeSetHandler) {
	t.target.handler = value
}
