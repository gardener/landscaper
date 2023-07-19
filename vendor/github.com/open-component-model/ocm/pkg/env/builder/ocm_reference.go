// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
)

type ocmReference struct {
	base

	meta compdesc.ComponentReference
}

const T_OCMREF = "reference"

func (r *ocmReference) Type() string {
	return T_OCMREF
}

func (r *ocmReference) Set() {
	r.Builder.ocm_meta = &r.meta.ElementMeta
}

func (r *ocmReference) Close() error {
	return r.ocm_vers.SetReference(&r.meta)
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Reference(name, comp, vers string, f ...func()) {
	b.expect(b.ocm_vers, T_OCMVERSION)
	r := &ocmReference{}
	r.meta.Name = name
	r.meta.Version = vers
	r.meta.ComponentName = comp
	b.configure(r, f)
}
