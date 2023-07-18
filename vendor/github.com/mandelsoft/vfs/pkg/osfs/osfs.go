/*
 * Copyright 2020 Mandelsoft. All rights reserved.
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

package osfs

import (
	"os"
	"time"

	"github.com/mandelsoft/filepath/pkg/filepath"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type osFileSystem struct {
}

func New() vfs.FileSystem {
	return &osFileSystem{}
}

func (osFileSystem) Name() string {
	return "OsFs"
}

func (osFileSystem) VolumeName(name string) string {
	return filepath.VolumeName(name)
}

func (osFileSystem) FSTempDir() string {
	return os.TempDir()
}

func (osFileSystem) Normalize(path string) string {
	return mapPath(path)
}

func (osFileSystem) Getwd() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return mapPath(d), nil
}

func (osFileSystem) Create(name string) (vfs.File, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	f, e := os.Create(name)
	if f == nil {
		return nil, e
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		f.Close()
		return nil, err
	}
	return utils.NewRenamedFile(abs, f), e
}

func (osFileSystem) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

func (osFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFileSystem) Open(name string) (vfs.File, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	f, e := os.Open(name)
	if f == nil {
		return nil, e
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		f.Close()
		return nil, err
	}
	return utils.NewRenamedFile(abs, f), e
}

func (osFileSystem) OpenFile(name string, flag int, perm os.FileMode) (vfs.File, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	f, e := os.OpenFile(name, flag, perm)
	if f == nil {
		return nil, e
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		f.Close()
		return nil, err
	}
	return utils.NewRenamedFile(abs, f), e
}

func (osFileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (osFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (osFileSystem) Rename(oldname, newname string) error {
	return os.Rename(oldname, newname)
}

func (osFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (osFileSystem) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (osFileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return os.Chtimes(name, atime, mtime)
}

func (osFileSystem) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (osFileSystem) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (osFileSystem) Readlink(name string) (string, error) {
	return os.Readlink(name)
}
