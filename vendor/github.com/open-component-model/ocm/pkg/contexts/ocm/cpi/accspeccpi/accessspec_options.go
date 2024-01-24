// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accspeccpi

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets/flagsetscheme"
)

type AccessSpecTypeOption = flagsetscheme.TypeOption

func WithFormatSpec(value string) AccessSpecTypeOption {
	return flagsetscheme.WithFormatSpec(value)
}

func WithDescription(value string) AccessSpecTypeOption {
	return flagsetscheme.WithDescription(value)
}

func WithConfigHandler(value flagsets.ConfigOptionTypeSetHandler) AccessSpecTypeOption {
	return flagsetscheme.WithConfigHandler(value)
}
