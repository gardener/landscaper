// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

const T_OCMVERSION = "component version"

type ocmVersion struct {
	base
	kind string
	cpi.ComponentVersionAccess
}

func (r *ocmVersion) Type() string {
	if r.kind != "" {
		return r.kind
	}
	return T_OCMVERSION
}

func (r *ocmVersion) Set() {
	r.Builder.ocm_vers = r.ComponentVersionAccess
	r.Builder.ocm_labels = &r.ComponentVersionAccess.GetDescriptor().Labels
}

func (r *ocmVersion) Close() error {
	list := errors.ErrListf("adding component version")
	if r.Builder.ocm_comp != nil {
		list.Add(r.Builder.ocm_comp.AddVersion(r.ComponentVersionAccess))
	}
	list.Add(r.ComponentVersionAccess.Close())
	return list.Result()
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Version(name string, f ...func()) {
	b.expect(b.ocm_comp, T_OCMCOMPONENT)
	v, err := b.ocm_comp.LookupVersion(name)
	if err != nil {
		if errors.IsErrNotFound(err) {
			v, err = b.ocm_comp.NewVersion(name)
		}
	}
	b.failOn(err)
	v.GetDescriptor().Provider.Name = metav1.ProviderName("ACME")
	b.configure(&ocmVersion{ComponentVersionAccess: v}, f)
}

func (b *Builder) ComponentVersion(name, version string, f ...func()) {
	b.Component(name, func() {
		b.Version(version, f...)
	})
}
