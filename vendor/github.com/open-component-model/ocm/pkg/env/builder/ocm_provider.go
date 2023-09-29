// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
)

func (b *Builder) Provider(name string) {
	b.expect(b.ocm_vers, T_OCMVERSION)
	b.ocm_vers.GetDescriptor().Provider.Name = metav1.ProviderName(name)
}

func (b *Builder) ProviderLabel(name string, value interface{}) {
	b.expect(b.ocm_vers, T_OCMVERSION)
	b.failOn(b.ocm_vers.GetDescriptor().Provider.Labels.Set(name, value))
}
