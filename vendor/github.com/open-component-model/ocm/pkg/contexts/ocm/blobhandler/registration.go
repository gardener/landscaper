// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blobhandler

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
)

func RegisterHandlerByName(ctx cpi.ContextProvider, name string, config HandlerConfig, opts ...HandlerOption) error {
	o, err := For(ctx).RegisterByName(name, ctx.OCMContext(), config, opts...)
	if err != nil {
		return err
	}
	if !o {
		return fmt.Errorf("no matching handler found for %q", name)
	}
	return nil
}

func WithPrio(prio int) HandlerOption {
	return cpi.WithPrio(prio)
}

func ForArtifactType(t string) HandlerOption {
	return cpi.ForArtifactType(t)
}

func ForMimeType(t string) HandlerOption {
	return cpi.ForMimeType(t)
}

func ForRepo(ctxtype string, repotype string) HandlerOption {
	return cpi.ForRepo(ctxtype, repotype)
}
