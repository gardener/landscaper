// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
)

const T_OCMCOMPONENT = "component"

type ocmComponent struct {
	base
	kind string
	cpi.ComponentAccess
}

func (r *ocmComponent) Type() string {
	if r.kind != "" {
		return r.kind
	}
	return T_OCMCOMPONENT
}

func (r *ocmComponent) Set() {
	r.Builder.ocm_comp = r.ComponentAccess
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Component(name string, f ...func()) {
	b.expect(b.ocm_repo, T_OCMREPOSITORY)
	c, err := b.ocm_repo.LookupComponent(name)
	b.failOn(err)
	b.configure(&ocmComponent{ComponentAccess: c}, f)
}
