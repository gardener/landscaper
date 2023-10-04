// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

const T_OCIREPOSITORY = "oci repository"

type ociRepository struct {
	base
	kind string
	cpi.Repository
}

func (r *ociRepository) Type() string {
	if r.kind != "" {
		return r.kind
	}
	return T_OCIREPOSITORY
}

func (r *ociRepository) Set() {
	r.Builder.oci_repo = r.Repository
}
