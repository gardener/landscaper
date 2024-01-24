// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

const AltType = "ctf"

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
	return MapReference(ctx, u)
}

func explicit(t string) bool {
	for _, f := range SupportedFormats() {
		if t == string(f) {
			return true
		}
	}
	return t == Type || t == AltType
}

func MapReference(ctx cpi.Context, u *cpi.UniformRepositorySpec) (cpi.RepositorySpec, error) {
	path := u.Info
	if u.Info == "" {
		if u.Host == "" || u.Type == "" {
			return nil, nil
		}
		path = u.Host
	}
	fs := vfsattr.Get(ctx)

	typ, _ := accessobj.MapType(u.Type, Type, accessio.FormatNone, true, AltType)
	hint, f := accessobj.MapType(u.TypeHint, Type, accessio.FormatDirectory, true, AltType)
	if !u.CreateIfMissing {
		hint = ""
	}
	create, ok, err := accessobj.CheckFile(Type, hint, explicit(accessio.TypeForTypeSpec(u.Type)), path, fs, ArtifactIndexFileName)
	if !ok || (err != nil && typ == "") {
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}
	}
	mode := accessobj.ACC_WRITABLE
	createHint := accessio.FormatNone
	if create {
		mode |= accessobj.ACC_CREATE
		createHint = f
	}
	return NewRepositorySpec(mode, path, createHint, accessio.PathFileSystem(fs))
}
