// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package transferhandler

import (
	"github.com/open-component-model/ocm/pkg/contexts/config"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/errors"
)

type TransferOptions interface{}

type TransferOption interface {
	ApplyTransferOption(TransferOptions) error
}

type TransferHandler interface {
	OverwriteVersion(src ocm.ComponentVersionAccess, tgt ocm.ComponentVersionAccess) (bool, error)

	TransferVersion(repo ocm.Repository, src ocm.ComponentVersionAccess, meta *compdesc.ComponentReference, tgt ocm.Repository) (ocm.ComponentVersionAccess, TransferHandler, error)
	TransferResource(src ocm.ComponentVersionAccess, a ocm.AccessSpec, r ocm.ResourceAccess) (bool, error)
	TransferSource(src ocm.ComponentVersionAccess, a ocm.AccessSpec, r ocm.SourceAccess) (bool, error)

	HandleTransferResource(r ocm.ResourceAccess, m ocm.AccessMethod, hint string, t ocm.ComponentVersionAccess) error
	HandleTransferSource(r ocm.SourceAccess, m ocm.AccessMethod, hint string, t ocm.ComponentVersionAccess) error
}

func ApplyOptions(set TransferOptions, opts ...TransferOption) error {
	list := errors.ErrListf("transfer options")
	for _, o := range opts {
		list.Add(o.ApplyTransferOption(set))
	}
	return list.Result()
}

func From(ctx config.ContextProvider, opts TransferOptions) error {
	_, err := ctx.ConfigContext().ApplyTo(-1, opts)
	return err
}

func BoolP(b bool) *bool {
	return &b
}

func AsBool(b *bool) bool {
	return b != nil && *b
}
