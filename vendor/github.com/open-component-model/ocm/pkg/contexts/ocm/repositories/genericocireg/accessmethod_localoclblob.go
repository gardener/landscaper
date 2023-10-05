// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
)

type localOCIBlobAccessMethod struct {
	*localBlobAccessMethod
}

var _ cpi.AccessMethod = (*localOCIBlobAccessMethod)(nil)

func newLocalOCIBlobAccessMethod(a *localblob.AccessSpec, ns oci.NamespaceAccess, art oci.ArtifactAccess) cpi.AccessMethod {
	return &localOCIBlobAccessMethod{
		localBlobAccessMethod: newLocalBlobAccessMethod(a, ns, art),
	}
}

func (m *localOCIBlobAccessMethod) MimeType() string {
	digest := digest.Digest(m.spec.LocalReference)
	desc := m.artifact.GetDescriptor().GetBlobDescriptor(digest)
	if desc == nil {
		return ""
	}
	return desc.MediaType
}
