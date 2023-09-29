// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
)

type ocmProvider struct {
	base

	provider compdesc.Provider
}

const T_OCMPROVIDER = "provider"

func (r *ocmProvider) Type() string {
	return T_OCMPROVIDER
}

func (r *ocmProvider) Set() {
	r.Builder.ocm_labels = &r.provider.Labels
}

func (r *ocmProvider) Close() error {
	r.ocm_vers.GetDescriptor().Provider = r.provider
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Provider(name string, f ...func()) {
	b.expect(b.ocm_vers, T_OCMVERSION)
	r := &ocmProvider{}
	r.provider.Name = metav1.ProviderName(name)
	b.configure(r, f)
}
