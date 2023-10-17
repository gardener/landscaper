// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signingattr

import (
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/signing"
)

func init() {
	datacontext.RegisterSetupHandler(datacontext.SetupHandlerFunction(setupContext))
}

func setupContext(mode datacontext.BuilderMode, ctx datacontext.Context) {
	if octx, ok := ctx.(Context); ok {
		switch mode {
		case datacontext.MODE_SHARED:
			fallthrough
		case datacontext.MODE_DEFAULTED:
			// do nothing, fallback to the default attribute lookup
		case datacontext.MODE_EXTENDED:
			Set(octx, signing.NewRegistry(signing.DefaultRegistry().HandlerRegistry(), signing.DefaultRegistry().KeyRegistry()))
		case datacontext.MODE_CONFIGURED:
			Set(octx, signing.DefaultRegistry().Copy())
		case datacontext.MODE_INITIAL:
			Set(octx, signing.NewRegistry(nil, nil))
		}
	}
}
