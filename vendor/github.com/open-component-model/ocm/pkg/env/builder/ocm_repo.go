// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	ocm "github.com/open-component-model/ocm/pkg/contexts/ocm/context"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ocireg"
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

func (b *Builder) OCMRepository(spec ocm.RepositorySpec, f ...func()) {
	repo, err := b.OCMContext().RepositoryForSpec(spec)
	b.failOn(err)
	b.configure(&ocmRepository{Repository: repo, kind: T_OCMREPOSITORY}, f)
}

func (b *Builder) OCIBasedOCMRepository(url string, path string, f ...func()) {
	spec := ocireg.NewRepositorySpec(url, &ocireg.ComponentRepositoryMeta{
		SubPath: path,
	})
	b.OCMRepository(spec, f...)
}
