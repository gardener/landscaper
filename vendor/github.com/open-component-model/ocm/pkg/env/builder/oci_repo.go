// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
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

func (b *Builder) GeneralOCIRepository(spec oci.RepositorySpec, f ...func()) {
	repo, err := b.OCIContext().RepositoryForSpec(spec)
	b.failOn(err)
	b.configure(&ociRepository{Repository: repo, kind: T_OCIREPOSITORY}, f)
}

func (b *Builder) OCIRegistry(url string, path string, f ...func()) {
	spec := ocireg.NewRepositorySpec(url)
	b.GeneralOCIRepository(spec, f...)
}
