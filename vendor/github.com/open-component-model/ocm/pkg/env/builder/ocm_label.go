// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
)

const T_OCMLABELS = "element with labels"

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Label(name string, value interface{}, opts ...metav1.LabelOption) {
	b.expect(b.ocm_labels, T_OCMLABELS)

	b.failOn(b.ocm_labels.Set(name, value, opts...))
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) RemoveLabel(name string) {
	b.expect(b.ocm_labels, T_OCMLABELS)

	b.ocm_labels.Remove(name)
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) ClearLabels() {
	b.expect(b.ocm_labels, T_OCMLABELS)

	*b.ocm_labels = nil
}
