// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"fmt"

	"github.com/mandelsoft/vfs/pkg/composefs"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/readonlyfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/config"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
)

////////////////////////////////////////////////////////////////////////////////

type Option interface {
	Mount(fs *composefs.ComposedFileSystem) error
	GetFilesystem() vfs.FileSystem
}

type dummyOption struct{}

var _ Option = (*dummyOption)(nil)

func (o dummyOption) GetFilesystem() vfs.FileSystem {
	return nil
}

func (dummyOption) Mount(*composefs.ComposedFileSystem) error {
	return nil
}

type fsOpt struct {
	dummyOption
	path string
	fs   vfs.FileSystem
}

func FileSystem(fs vfs.FileSystem, path string) fsOpt {
	return fsOpt{
		path: path,
		fs:   fs,
	}
}

func (o fsOpt) GetFilesystem() vfs.FileSystem {
	if o.path == "" {
		return o.fs
	}
	return nil
}

func (o fsOpt) Mount(cfs *composefs.ComposedFileSystem) error {
	if o.path == "" {
		return nil
	}
	return cfs.Mount(o.path, o.fs)
}

type tdOpt struct {
	dummyOption
	path       string
	source     string
	modifiable bool
}

func TestData(paths ...string) tdOpt {
	path := "/testdata"
	source := "testdata"

	switch len(paths) {
	case 0:
	case 1:
		source = paths[0]
	case 2:
		source = paths[0]
		path = paths[1]
	default:
		panic("invalid number of arguments")
	}
	return tdOpt{
		path:   path,
		source: source,
	}
}

func ModifiableTestData(paths ...string) tdOpt {
	path := "/testdata"
	source := "testdata"

	switch len(paths) {
	case 0:
	case 1:
		source = paths[0]
	case 2:
		source = paths[0]
		path = paths[1]
	default:
		panic("invalid number of arguments")
	}
	return tdOpt{
		path:       path,
		source:     source,
		modifiable: true,
	}
}

func (o tdOpt) Mount(cfs *composefs.ComposedFileSystem) error {
	fs, err := projectionfs.New(osfs.New(), o.source)
	if err != nil {
		return fmt.Errorf("faild to create new project fs: %w", err)
	}

	if o.modifiable {
		fs = layerfs.New(memoryfs.New(), fs)
	} else {
		fs = readonlyfs.New(fs)
	}

	if err = cfs.MkdirAll(o.path, vfs.ModePerm); err != nil {
		return err
	}

	if err := cfs.Mount(o.path, fs); err != nil {
		return fmt.Errorf("faild to mount cfs: %w", err)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type Environment struct {
	vfs.VFS
	ctx        ocm.Context
	filesystem *composefs.ComposedFileSystem
}

func NewEnvironment(opts ...Option) *Environment {
	var basefs vfs.FileSystem

	for _, o := range opts {
		fs := o.GetFilesystem()
		if fs != nil {
			basefs = fs
		}
	}
	if basefs == nil {
		tmpfs, err := osfs.NewTempFileSystem()
		if err != nil {
			panic(err)
		}
		basefs = tmpfs
		defer func() {
			vfs.Cleanup(basefs)
		}()
	}
	err := basefs.Mkdir("/tmp", vfs.ModePerm)
	if err != nil {
		panic(err)
	}
	fs := composefs.New(basefs, "/tmp")
	for _, o := range opts {
		err := o.Mount(fs)
		if err != nil {
			panic(err)
		}
	}
	ctx := ocm.WithCredentials(credentials.WithConfigs(config.New()).New()).New()
	vfsattr.Set(ctx.AttributesContext(), fs)
	basefs = nil
	return &Environment{
		VFS:        vfs.New(fs),
		ctx:        ctx,
		filesystem: fs,
	}
}

var _ accessio.Option = (*Environment)(nil)

func (e *Environment) ApplyOption(options accessio.Options) error {
	options.SetPathFileSystem(e.FileSystem())
	return nil
}

func (e *Environment) OCMContext() ocm.Context {
	return e.ctx
}

func (e *Environment) OCIContext() oci.Context {
	return e.ctx.OCIContext()
}

func (e *Environment) CredentialsContext() credentials.Context {
	return e.ctx.CredentialsContext()
}

func (e *Environment) ConfigContext() config.Context {
	return e.ctx.ConfigContext()
}

func (e *Environment) FileSystem() vfs.FileSystem {
	return vfsattr.Get(e.ctx)
}
