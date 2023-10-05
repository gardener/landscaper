// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg"
)

const T_OCMREPOSITORY = "ocm repository"

type ocmRepository struct {
	base
	kind string
	cpi.Repository
}

func (r *ocmRepository) Type() string {
	if r.kind != "" {
		return r.kind
	}
	return T_OCMREPOSITORY
}

func (r *ocmRepository) Set() {
	r.Builder.ocm_repo = r.Repository
	r.Builder.oci_repo = genericocireg.GetOCIRepository(r.Repository)
}
