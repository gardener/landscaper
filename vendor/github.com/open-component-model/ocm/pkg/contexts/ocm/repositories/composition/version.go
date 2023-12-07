// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package composition

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/refmgmt"
)

func NewComponentVersion(ctx cpi.ContextProvider, name, vers string) cpi.ComponentVersionAccess {
	repo := NewRepository(ctx)
	defer repo.Close()
	if !refmgmt.Lazy(repo) {
		panic("wrong composition repo implementation")
	}
	c, err := repo.LookupComponent(name)
	if err != nil {
		panic("wrong composition repo implementation: " + err.Error())
	}
	defer c.Close()
	cv, err := c.NewVersion(vers)
	if err != nil {
		panic("wrong composition repo implementation: " + err.Error())
	}
	return cv
}
