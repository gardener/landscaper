// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/comparch"
)

const T_COMPARCH = "component archive"

func (b *Builder) ComponentArchive(path string, fmt accessio.FileFormat, name, vers string, f ...func()) {
	r, err := comparch.Open(b.OCMContext(), accessobj.ACC_WRITABLE|accessobj.ACC_CREATE, path, 0o777, accessio.PathFileSystem(b.FileSystem()))
	b.failOn(err)
	r.SetName(name)
	r.SetVersion(vers)
	r.GetDescriptor().Provider.Name = metav1.ProviderName("ACME")

	b.configure(&ocmVersion{ComponentVersionAccess: r, kind: T_COMPARCH}, f)
}
