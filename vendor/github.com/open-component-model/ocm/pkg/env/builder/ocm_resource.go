// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
)

type ocmResource struct {
	base

	meta   compdesc.ResourceMeta
	access compdesc.AccessSpec
	blob   accessio.BlobAccess
	hint   string
}

const T_OCMRESOURCE = "resource"

func (r *ocmResource) Type() string {
	return T_OCMRESOURCE
}

func (r *ocmResource) Set() {
	r.Builder.ocm_rsc = &r.meta
	r.Builder.ocm_acc = &r.access
	r.Builder.ocm_meta = &r.meta.ElementMeta
	r.Builder.blob = &r.blob
	r.Builder.hint = &r.hint
}

func (r *ocmResource) Close() error {
	switch {
	case r.access != nil:
		return r.Builder.ocm_vers.SetResource(&r.meta, r.access)
	case r.blob != nil:
		return r.Builder.ocm_vers.SetResourceBlob(&r.meta, r.blob, r.hint, nil)
	}
	return errors.New("access or blob required")
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Resource(name, vers, typ string, relation metav1.ResourceRelation, f ...func()) {
	b.expect(b.ocm_vers, T_OCMVERSION)
	r := &ocmResource{}
	r.meta.Name = name
	r.meta.Version = vers
	r.meta.Type = typ
	r.meta.Relation = relation
	b.configure(r, f)
}
