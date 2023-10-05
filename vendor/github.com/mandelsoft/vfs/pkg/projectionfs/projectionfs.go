/*
 * Copyright 2022 Mandelsoft. All rights reserved.
 *  This file is licensed under the Apache Software License, v. 2 except as noted
 *  otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package projectionfs

import (
	"fmt"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

// RootPath is the interface for projected filesystems
// to determine the root folder in the underlying
// filesystem.
type RootPath interface {
	Root() string
}

func Root(fs vfs.FileSystem) string {
	if r, ok := fs.(RootPath); ok {
		return r.Root()
	}
	return ""
}

type ProjectionFileSystem struct {
	*utils.MappedFileSystem
	projection string
}

type adapter struct {
	fs *ProjectionFileSystem
}

func (a *adapter) MapPath(name string) (vfs.FileSystem, string) {
	return a.fs.Base(), vfs.Join(a.fs.Base(), a.fs.projection, name)
}

func New(base vfs.FileSystem, path string) (vfs.FileSystem, error) {
	eff, err := vfs.Canonical(base, path, true)
	if err != nil {
		return nil, err
	}
	fs := &ProjectionFileSystem{projection: eff}
	fs.MappedFileSystem = utils.NewMappedFileSystem(base, &adapter{fs})
	return fs, nil
}

func (p *ProjectionFileSystem) Name() string {
	return fmt.Sprintf("ProjectionFilesytem [%s]%s", p.Base().Name(), p.projection)
}

func (p *ProjectionFileSystem) Projection() string {
	return p.projection
}

func (p *ProjectionFileSystem) Root() string {
	if r, ok := p.Base().(RootPath); ok {
		return vfs.Join(p, r.Root(), p.projection)
	}
	return p.projection
}
