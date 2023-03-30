// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
)

////////////////////////////////////////////////////////////////////////////////
// Access Type Options

type AccessSpecOptionTarget interface {
	SetFormat(string)
	SetDescription(string)
	SetConfigHandler(flagsets.ConfigOptionTypeSetHandler)
}

type AccessSpecTypeOption interface {
	ApplyToAccessSpecOptionTarget(AccessSpecOptionTarget)
}

////////////////////////////////////////////////////////////////////////////////

type formatOption struct {
	value string
}

func WithFormatSpec(value string) AccessSpecTypeOption {
	return formatOption{value}
}

func (o formatOption) ApplyToAccessSpecOptionTarget(t AccessSpecOptionTarget) {
	t.SetFormat(o.value)
}

////////////////////////////////////////////////////////////////////////////////

type descriptionOption struct {
	value string
}

func WithDescription(value string) AccessSpecTypeOption {
	return descriptionOption{value}
}

func (o descriptionOption) ApplyToAccessSpecOptionTarget(t AccessSpecOptionTarget) {
	t.SetDescription(o.value)
}

////////////////////////////////////////////////////////////////////////////////

type configOption struct {
	value flagsets.ConfigOptionTypeSetHandler
}

func WithConfigHandler(value flagsets.ConfigOptionTypeSetHandler) AccessSpecTypeOption {
	return configOption{value}
}

func (o configOption) ApplyToAccessSpecOptionTarget(t AccessSpecOptionTarget) {
	t.SetConfigHandler(o.value)
}
