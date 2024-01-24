// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsetscheme

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/optionutils"
	"github.com/open-component-model/ocm/pkg/runtime/descriptivetype"
)

////////////////////////////////////////////////////////////////////////////////
// Access Type Options

type OptionTarget interface {
	descriptivetype.OptionTarget
	SetConfigHandler(flagsets.ConfigOptionTypeSetHandler)
}

type TypeOption = optionutils.Option[OptionTarget]

////////////////////////////////////////////////////////////////////////////////
// options derived from descriptivetype.

func WithFormatSpec(value string) TypeOption {
	return optionutils.MapOptionTarget[OptionTarget](descriptivetype.WithFormatSpec(value))
}

func WithDescription(value string) TypeOption {
	return optionutils.MapOptionTarget[OptionTarget](descriptivetype.WithDescription(value))
}

////////////////////////////////////////////////////////////////////////////////
// additional options.

type configOption struct {
	value flagsets.ConfigOptionTypeSetHandler
}

func WithConfigHandler(value flagsets.ConfigOptionTypeSetHandler) TypeOption {
	return configOption{value}
}

func (o configOption) ApplyTo(t OptionTarget) {
	t.SetConfigHandler(o.value)
}
