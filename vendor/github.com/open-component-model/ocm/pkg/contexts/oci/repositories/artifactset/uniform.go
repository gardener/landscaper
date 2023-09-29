// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactset

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

func init() {
	h := &repospechandler{}
	cpi.RegisterRepositorySpecHandler(h, "")
	cpi.RegisterRepositorySpecHandler(h, Type)
	for _, f := range SupportedFormats() {
		cpi.RegisterRepositorySpecHandler(h, string(f))
	}
}

type repospechandler struct{}

func (h *repospechandler) MapReference(ctx cpi.Context, u *cpi.UniformRepositorySpec) (cpi.RepositorySpec, error) {
	path := u.Info
	if u.Info == "" {
		if u.Host == "" || u.Type == "" {
			return nil, nil
		}
		path = u.Host
	}
	fs := vfsattr.Get(ctx)

	hint := u.TypeHint
	if !u.CreateIfMissing {
		hint = ""
	}

	create, ok, err := accessobj.CheckFile(Type, hint, accessio.TypeForType(u.Type) == Type, path, fs, ArtifactSetDescriptorFileName)
	if err == nil && !ok {
		create, ok, err = accessobj.CheckFile(Type, hint, accessio.TypeForType(u.Type) == Type, path, fs, OCIArtifactSetDescriptorFileName)
	}

	if !ok || err != nil {
		return nil, err
	}

	mode := accessobj.ACC_WRITABLE
	if create {
		mode |= accessobj.ACC_CREATE
	}
	return NewRepositorySpec(mode, path, accessio.FileFormatForType(u.Type), accessio.PathFileSystem(fs))
}
