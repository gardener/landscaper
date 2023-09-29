// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

func (b *Builder) Tags(tags ...string) {
	b.expect(b.oci_tags, T_OCIARTIFACT)
	*b.oci_tags = append(*b.oci_tags, tags...)
}
