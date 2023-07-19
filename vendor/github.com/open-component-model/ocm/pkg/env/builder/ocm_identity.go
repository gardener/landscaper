// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) ExtraIdentity(name string, value string) {
	b.expect(b.ocm_meta, T_OCMMETA)

	b.ocm_meta.ExtraIdentity.Set(name, value)
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) RemoveExtraIdentity(name string) {
	b.expect(b.ocm_meta, T_OCMMETA)

	b.ocm_meta.ExtraIdentity.Remove(name)
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) ClearExtraIdentities() {
	b.expect(b.ocm_meta, T_OCMMETA)

	b.ocm_meta.ExtraIdentity = nil
}
