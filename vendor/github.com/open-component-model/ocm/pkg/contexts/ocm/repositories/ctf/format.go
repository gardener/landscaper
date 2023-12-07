// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ctf"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg"
)

var (
	FormatDirectory = ctf.FormatDirectory
	FormatTAR       = ctf.FormatTAR
	FormatTGZ       = ctf.FormatTGZ
)

type Object = ctf.Object

type FormatHandler = ctf.FormatHandler

////////////////////////////////////////////////////////////////////////////////

func GetFormats() []string {
	return ctf.GetFormats()
}

func GetFormat(name accessio.FileFormat) FormatHandler {
	return ctf.GetFormat(name)
}

////////////////////////////////////////////////////////////////////////////////

const (
	ACC_CREATE   = accessobj.ACC_CREATE
	ACC_WRITABLE = accessobj.ACC_WRITABLE
	ACC_READONLY = accessobj.ACC_READONLY
)

func Open(ctx cpi.ContextProvider, acc accessobj.AccessMode, path string, mode vfs.FileMode, opts ...accessio.Option) (cpi.Repository, error) {
	r, err := ctf.Open(cpi.FromProvider(ctx), acc, path, mode, opts...)
	if err != nil {
		return nil, err
	}
	return genericocireg.NewRepository(cpi.FromProvider(ctx), nil, r), nil
}

func Create(ctx cpi.ContextProvider, acc accessobj.AccessMode, path string, mode vfs.FileMode, opts ...accessio.Option) (cpi.Repository, error) {
	r, err := ctf.Create(cpi.FromProvider(ctx), acc, path, mode, opts...)
	if err != nil {
		return nil, err
	}
	return genericocireg.NewRepository(cpi.FromProvider(ctx), nil, r), nil
}

////////////////////////////////////////////////////////////////////////////////

type CTFOptions struct {
	genericocireg.ComponentRepositoryMeta
}

type CTFOption interface {
	accessio.Option
	ApplyCTFOption(opts *CTFOptions)
}

// RepositoryPrefix set the OCI repository prefix used to store the component
// versions.
func RepositoryPrefix(path string) accessio.Option {
	return &optPrefix{path}
}

type optPrefix struct {
	prefix string
}

var _ CTFOption = (*optPrefix)(nil)

// ApplyOption does nothing, because this is no standard option.
func (o *optPrefix) ApplyOption(options accessio.Options) error {
	return nil
}

func (o *optPrefix) ApplyCTFOption(opts *CTFOptions) {
	opts.SubPath = o.prefix
}
