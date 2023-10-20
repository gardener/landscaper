// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package clitypes

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
)

////////////////////////////////////////////////////////////////////////////////
// Access Type Options

type CLIOptionTarget interface {
	SetFormat(string)
	SetDescription(string)
	SetConfigHandler(flagsets.ConfigOptionTypeSetHandler)
}

type CLITypeOption interface {
	ApplyToCLIOptionTarget(CLIOptionTarget)
}

////////////////////////////////////////////////////////////////////////////////

type formatOption struct {
	value string
}

func WithFormatSpec(value string) CLITypeOption {
	return formatOption{value}
}

func (o formatOption) ApplyToCLIOptionTarget(t CLIOptionTarget) {
	t.SetFormat(o.value)
}

////////////////////////////////////////////////////////////////////////////////

type descriptionOption struct {
	value string
}

func WithDescription(value string) CLITypeOption {
	return descriptionOption{value}
}

func (o descriptionOption) ApplyToCLIOptionTarget(t CLIOptionTarget) {
	t.SetDescription(o.value)
}

////////////////////////////////////////////////////////////////////////////////

type configOption struct {
	value flagsets.ConfigOptionTypeSetHandler
}

func WithConfigHandler(value flagsets.ConfigOptionTypeSetHandler) CLITypeOption {
	return configOption{value}
}

func (o configOption) ApplyToCLIOptionTarget(t CLIOptionTarget) {
	t.SetConfigHandler(o.value)
}
