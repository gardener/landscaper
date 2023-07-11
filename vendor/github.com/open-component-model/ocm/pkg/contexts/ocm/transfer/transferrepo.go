// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer/transferhandler"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer/transferhandler/standard"
	"github.com/open-component-model/ocm/pkg/errors"
)

func TransferComponents(printer common.Printer, closure TransportClosure, repo ocm.Repository, prefix string, all bool, tgt ocm.Repository, handler transferhandler.TransferHandler) error {
	if closure == nil {
		closure = TransportClosure{}
	}

	lister := repo.ComponentLister()
	if lister == nil {
		return errors.ErrNotSupported("ComponentLister")
	}
	if handler == nil {
		handler = standard.NewDefaultHandler(nil)
	}
	comps, err := lister.GetComponents(prefix, all)
	if err != nil {
		return err
	}
	list := errors.ErrListf("component transport")
	for _, c := range comps {
		transferVersions(common.AssurePrinter(printer), closure, list, handler, repo, c, tgt)
	}
	return list.Result()
}

func transferVersions(printer common.Printer, closure TransportClosure, list *errors.ErrorList, handler transferhandler.TransferHandler, repo ocm.Repository, c string, tgt ocm.Repository) {
	comp, err := repo.LookupComponent(c)
	if list.Addf(printer, err, "component %s", c) == nil {
		defer comp.Close()
		printer.Printf("transferring component %q...\n", c)
		subp := printer.AddGap("  ")
		vers, err := comp.ListVersions()

		if list.Addf(subp, err, "list versions for %s", c) == nil {
			for _, v := range vers {
				ref := compdesc.NewComponentReference("", c, v, nil)
				sub, h, err := handler.TransferVersion(repo, nil, ref, tgt)
				if list.Addf(subp, err, "version %s", v) != nil {
					continue
				}
				if sub != nil {
					if list.Addf(subp, err, "version %s", v) == nil {
						list.Addf(subp, TransferVersion(subp, closure, sub, tgt, h), "")
					}
					sub.Close()
				}
			}
		}
	}
}
