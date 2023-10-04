// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	. "github.com/onsi/gomega"
)

const T_OCMMETA = "element with metadata"

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Label(name string, value interface{}) {
	b.expect(b.ocm_meta, T_OCMMETA)

	ExpectWithOffset(1, b.ocm_meta.Labels.Set(name, value)).To(Succeed())
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) RemoveLabel(name string) {
	b.expect(b.ocm_meta, T_OCMMETA)

	b.ocm_meta.Labels.Remove(name)
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) ClearLabels() {
	b.expect(b.ocm_meta, T_OCMMETA)

	b.ocm_meta.Labels = nil
}
