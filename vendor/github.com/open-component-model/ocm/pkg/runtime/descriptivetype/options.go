// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package descriptivetype

import (
	"github.com/open-component-model/ocm/pkg/optionutils"
	"github.com/open-component-model/ocm/pkg/runtime"
)

////////////////////////////////////////////////////////////////////////////////

// TypeObjectTarget is used as target for option functions, it provides
// setters for fields, which should not be modifiable for a final type object.
type TypeObjectTarget[E runtime.VersionedTypedObject] struct {
	target *TypedObjectTypeObject[E]
}

func NewTypeObjectTarget[E runtime.VersionedTypedObject](target *TypedObjectTypeObject[E]) *TypeObjectTarget[E] {
	return &TypeObjectTarget[E]{target}
}

func (t *TypeObjectTarget[E]) SetDescription(value string) {
	t.target.description = value
}

func (t *TypeObjectTarget[E]) SetFormat(value string) {
	t.target.format = value
}

////////////////////////////////////////////////////////////////////////////////
// Descriptive Type Options

type OptionTarget interface {
	SetFormat(string)
	SetDescription(string)
}

type Option = optionutils.Option[OptionTarget]

////////////////////////////////////////////////////////////////////////////////

type formatOption struct {
	value string
}

func WithFormatSpec(value string) Option {
	return formatOption{value}
}

func (o formatOption) ApplyTo(t OptionTarget) {
	t.SetFormat(o.value)
}

////////////////////////////////////////////////////////////////////////////////

type descriptionOption struct {
	value string
}

func WithDescription(value string) Option {
	return descriptionOption{value}
}

func (o descriptionOption) ApplyTo(t OptionTarget) {
	t.SetDescription(o.value)
}
