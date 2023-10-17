// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/clitypes"
)

type AccessSpecTypeOption = clitypes.CLITypeOption

func WithFormatSpec(value string) AccessSpecTypeOption {
	return clitypes.WithFormatSpec(value)
}

func WithDescription(value string) AccessSpecTypeOption {
	return clitypes.WithDescription(value)
}

func WithConfigHandler(value flagsets.ConfigOptionTypeSetHandler) AccessSpecTypeOption {
	return clitypes.WithConfigHandler(value)
}
