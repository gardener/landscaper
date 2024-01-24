// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/errors"
)

const T_OCILAYER = "oci layer"

type ociLayer struct {
	base
	blob blobaccess.BlobAccess
}

func (r *ociLayer) Type() string {
	return T_OCILAYER
}

func (r *ociLayer) Set() {
	r.Builder.blob = &r.blob
}

func (r *ociLayer) Close() error {
	if r.blob == nil {
		return errors.Newf("config blob required")
	}
	m := r.Builder.oci_artacc.ManifestAccess()

	if r.oci_cleanuplayers {
		m.GetDescriptor().Layers = nil
		r.oci_cleanuplayers = false
	}
	_, err := m.AddLayer(r.blob, nil)
	if err == nil {
		r.result = artdesc.DefaultBlobDescriptor(r.blob)
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) Layer(f ...func()) *artdesc.Descriptor {
	b.expect(b.oci_artacc, T_OCIMANIFEST, func() bool { return b.oci_artacc.IsManifest() })
	return b.configure(&ociLayer{}, f).(*artdesc.Descriptor)
}
