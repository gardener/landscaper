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

package composefs

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type ComposedFileSystem struct {
	*utils.MappedFileSystem
	mounts  map[string]vfs.FileSystem
	tempdir string
}

type adapter struct {
	fs *ComposedFileSystem
}

func (a *adapter) MapPath(path string) (vfs.FileSystem, string) {
	var mountp string
	var mountfs vfs.FileSystem

	for p, fs := range a.fs.mounts {
		if p == path {
			return fs, vfs.PathSeparatorString
		}

		if strings.HasPrefix(path, p+vfs.PathSeparatorString) {
			if len(mountp) < len(p) {
				mountp = p
				mountfs = fs
			}
		}
	}
	if mountfs == nil {
		return a.fs.Base(), path
	}
	return mountfs, path[len(mountp):]
}

func New(root vfs.FileSystem, temp ...string) *ComposedFileSystem {
	tempdir := "/"
	if len(temp) > 0 && temp[0] != "" {
		if vfs.IsAbs(nil, temp[0]) {
			tempdir = temp[0]
		} else {
			tempdir = vfs.PathSeparatorString + temp[0]
		}
		tempdir = vfs.Trim(nil, tempdir)
	}
	fs := &ComposedFileSystem{mounts: map[string]vfs.FileSystem{}, tempdir: tempdir}
	fs.MappedFileSystem = utils.NewMappedFileSystem(root, &adapter{fs})
	return fs
}

func (c *ComposedFileSystem) Cleanup() error {
	var err error
	for _, m := range c.mounts {
		terr := vfs.Cleanup(m)
		if terr != nil {
			err = terr
		}
	}
	terr := c.MappedFileSystem.Cleanup()
	if terr != nil {
		return terr
	}
	return err
}

func (c *ComposedFileSystem) Name() string {
	return fmt.Sprintf("ComposedFileSystem [%s]", c.Base())
}

func (c *ComposedFileSystem) FSTempDir() string {
	return c.tempdir
}

func (c *ComposedFileSystem) MountTempDir(path string) (string, error) {
	if !vfs.IsAbs(nil, path) {
		path = vfs.PathSeparatorString + path
	}
	path = vfs.Trim(nil, path)
	dir, err := osfs.NewTempFileSystem()
	if err == nil {
		err = c.Mount(path, dir)
	}
	if err != nil {
		vfs.Cleanup(dir)
		return "", err
	}
	c.tempdir = path
	return path, err
}

func (c *ComposedFileSystem) MountTempFileSysten(path string, fs vfs.FileSystem) (string, error) {
	if !vfs.IsAbs(nil, path) {
		path = vfs.PathSeparatorString + path
	}
	path = vfs.Trim(nil, path)
	err := c.Mount(path, fs)
	if err != nil {
		return "", err
	}
	c.tempdir = path
	return path, err
}

func (c *ComposedFileSystem) Mount(path string, fs vfs.FileSystem) error {
	mountp, err := vfs.Canonical(c, path, true)
	if err != nil {
		return fmt.Errorf("mount failed: %s", err)
	}
	fi, err := c.Lstat(mountp)
	if err != nil {
		return fmt.Errorf("mount failed: %s", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("mount failed: mount point %s must be dir", mountp)
	}
	for p := range c.mounts {
		if p == mountp || strings.Contains(p, mountp+vfs.PathSeparatorString) {
			delete(c.mounts, p)
		}
	}
	c.mounts[mountp] = fs
	return nil
}
