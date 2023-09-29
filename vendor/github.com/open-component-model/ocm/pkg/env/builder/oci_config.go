// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/errors"
)

const T_OCICONFIG = "oci config"

type ociConfig struct {
	base
	blob accessio.BlobAccess
}

func (r *ociConfig) Type() string {
	return T_OCICONFIG
}

func (r *ociConfig) Set() {
	r.Builder.blob = &r.blob
}

func (r *ociConfig) Close() error {
	if r.blob == nil {
		return errors.Newf("config blob required")
	}
	m := r.Builder.oci_artacc.ManifestAccess()
	err := m.AddBlob(r.blob)
	if err != nil {
		return errors.Newf("cannot add config blob: %s", err)
	}
	d := artdesc.DefaultBlobDescriptor(r.blob)
	m.GetDescriptor().Config = *d
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Config(f ...func()) {
	b.expect(b.oci_artacc, T_OCIMANIFEST, func() bool { return b.oci_artacc.IsManifest() })
	b.configure(&ociConfig{}, f)
}
