// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package hpi

import (
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/internal"
)

func init() {
	datacontext.RegisterSetupHandler(datacontext.SetupHandlerFunction(setupContext))
}

func setupContext(mode datacontext.BuilderMode, ctx datacontext.Context) {
	if octx, ok := ctx.(cpi.Context); ok {
		switch mode {
		case datacontext.MODE_SHARED:
			fallthrough
		case datacontext.MODE_DEFAULTED:
			// do nothing, fallback to the default attribute lookup
		case datacontext.MODE_EXTENDED:
			SetFor(octx, NewRegistry(internal.DefaultRegistry))
		case datacontext.MODE_CONFIGURED:
			SetFor(octx, internal.DefaultRegistry.Copy())
		case datacontext.MODE_INITIAL:
			SetFor(octx, NewRegistry())
		}
	}
}
