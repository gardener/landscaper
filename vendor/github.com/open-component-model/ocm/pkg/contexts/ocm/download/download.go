// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/optionutils"
	"github.com/open-component-model/ocm/pkg/utils"
)

type Option = optionutils.Option[*Options]

type Options struct {
	Printer    common.Printer
	FileSystem vfs.FileSystem
}

func (o *Options) ApplyTo(opts *Options) {
	if o.Printer != nil {
		opts.Printer = o.Printer
	}
	if o.FileSystem != nil {
		opts.FileSystem = o.FileSystem
	}
}

////////////////////////////////////////////////////////////////////////////////

type filesystem struct {
	fs vfs.FileSystem
}

func (o *filesystem) ApplyTo(opts *Options) {
	if o.fs != nil {
		opts.FileSystem = o.fs
	}
}

func WithFileSystem(fs vfs.FileSystem) Option {
	return &filesystem{fs}
}

////////////////////////////////////////////////////////////////////////////////

type printer struct {
	pr common.Printer
}

func (o *printer) ApplyTo(opts *Options) {
	if o.pr != nil {
		opts.Printer = o.pr
	}
}

func WithPrinter(pr common.Printer) Option {
	return &printer{pr}
}

////////////////////////////////////////////////////////////////////////////////

func DownloadResource(ctx cpi.ContextProvider, r cpi.ResourceAccess, path string, opts ...Option) (string, error) {
	eff := optionutils.EvalOptions(opts...)

	fs := utils.FileSystem(eff.FileSystem)
	pr := utils.OptionalDefaulted(common.NewPrinter(nil), eff.Printer)
	_, tgt, err := For(ctx).Download(pr, r, path, fs)
	return tgt, err
}
