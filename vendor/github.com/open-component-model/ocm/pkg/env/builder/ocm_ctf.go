// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ctf"
)

const T_OCM_CTF = "ocm common transport format"

func (b *Builder) OCMCommonTransport(path string, fmt accessio.FileFormat, f ...func()) {
	r, err := ctf.Open(b.OCMContext(), accessobj.ACC_WRITABLE|accessobj.ACC_CREATE, path, 0o777, fmt, accessio.PathFileSystem(b.FileSystem()))
	b.failOn(err)
	b.configure(&ocmRepository{Repository: r, kind: T_OCM_CTF}, f)
}
