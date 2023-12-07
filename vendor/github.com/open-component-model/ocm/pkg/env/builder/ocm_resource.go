// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
)

type ocmResource struct {
	base

	orig   ocm.AccessSpec
	meta   compdesc.ResourceMeta
	access compdesc.AccessSpec
	blob   blobaccess.BlobAccess
	opts   ocm.ModificationOptions
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
	r.Builder.ocm_labels = &r.meta.ElementMeta.Labels
	r.Builder.ocm_modopts = &r.opts
	r.Builder.blob = &r.blob
	r.Builder.hint = &r.hint
}

func (r *ocmResource) Close() error {
	if r.orig != nil && (r.access == nil && r.blob == nil) {
		r.access = r.orig
	}
	switch {
	case r.access != nil:
		return r.Builder.ocm_vers.SetResource(&r.meta, r.access, r.opts.ApplyModificationOptions((ocm.ModifyResource())))
	case r.blob != nil:
		return r.Builder.ocm_vers.SetResourceBlob(&r.meta, r.blob, r.hint, nil, r.opts.ApplyModificationOptions((ocm.ModifyResource())))
	}
	return errors.New("access or blob required")
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Resource(name, vers, typ string, relation metav1.ResourceRelation, f ...func()) {
	b.expect(b.ocm_vers, T_OCMVERSION)
	r := &ocmResource{opts: b.def_modopts}
	r.meta.Name = name
	r.meta.Version = vers
	r.meta.Type = typ
	r.meta.Relation = relation
	b.configure(r, f)
}

func (b *Builder) ModifyResource(id metav1.Identity, f ...func()) {
	b.expect(b.ocm_vers, T_OCMVERSION)
	ra, err := b.ocm_vers.GetResource(id)
	b.failOn(err, 1)
	acc, err := ra.Access()
	b.failOn(err, 1)
	r := &ocmResource{orig: acc, opts: b.def_modopts, meta: *ra.Meta()}
	b.configure(r, f)
}

func (b *Builder) Digest(value, algo, norm string) {
	b.expect(b.ocm_rsc, T_OCMRESOURCE)
	b.ocm_rsc.Digest = &metav1.DigestSpec{
		HashAlgorithm:          algo,
		NormalisationAlgorithm: norm,
		Value:                  value,
	}
}

func (b *Builder) ModificationOptions(opts ...ocm.ModificationOption) {
	target := &b.def_modopts
	if b.ocm_modopts != nil {
		target = b.ocm_modopts
	}
	b.expect(target, "resource")
	for _, o := range opts {
		o.ApplyModificationOption(target)
	}
}
