// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

func (b *Builder) Annotation(name, value string) {
	b.expect(b.oci_annofunc, T_OCIARTIFACT+" or "+T_OCIARTIFACTSET)
	b.oci_annofunc(name, value)
}
