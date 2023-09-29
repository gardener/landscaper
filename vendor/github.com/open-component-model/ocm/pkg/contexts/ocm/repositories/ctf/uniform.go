// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ctf"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg"
)

func SupportedFormats() []accessio.FileFormat {
	return ctf.SupportedFormats()
}

func init() {
	h := &repospechandler{}
	cpi.RegisterRepositorySpecHandler(h, "")
	cpi.RegisterRepositorySpecHandler(h, ctf.Type)
	cpi.RegisterRepositorySpecHandler(h, "ctf")
	for _, f := range SupportedFormats() {
		cpi.RegisterRepositorySpecHandler(h, string(f))
		cpi.RegisterRepositorySpecHandler(h, "ctf+"+string(f))
		cpi.RegisterRepositorySpecHandler(h, ctf.Type+"+"+string(f))
	}
}

type repospechandler struct{}

func (h *repospechandler) MapReference(ctx cpi.Context, u *cpi.UniformRepositorySpec) (cpi.RepositorySpec, error) {
	if u.Info == "" {
		if u.Host == "" || u.Type == "" {
			return nil, nil
		}
	}
	spec, err := ctf.MapReference(ctx.OCIContext(), &oci.UniformRepositorySpec{
		Type:            u.Type,
		Host:            u.Host,
		Info:            u.Info,
		CreateIfMissing: u.CreateIfMissing,
		TypeHint:        u.TypeHint,
	})
	if err != nil || spec == nil {
		return nil, err
	}
	return genericocireg.NewRepositorySpec(spec, nil), nil
}
