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

package readonlyfs

import (
	"os"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type readonlyFileSystem struct {
	vfs.FileSystem
}

var _ vfs.FileSystem = &readonlyFileSystem{}

func New(fs vfs.FileSystem) vfs.FileSystem {
	return &readonlyFileSystem{fs}
}

func (r *readonlyFileSystem) Mkdir(path string, perm os.FileMode) error {
	return ErrReadOnly
}

func (r *readonlyFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return ErrReadOnly
}

func (r *readonlyFileSystem) Create(path string) (vfs.File, error) {
	return nil, ErrReadOnly
}

func (r *readonlyFileSystem) OpenFile(path string, flags int, perm os.FileMode) (vfs.File, error) {
	if flags&(os.O_WRONLY|os.O_CREATE|os.O_RDWR) != 0 {
		return nil, ErrReadOnly
	}
	return r.FileSystem.OpenFile(path, flags, perm)
}

func (r *readonlyFileSystem) Symlink(oldname, newname string) error {
	return ErrReadOnly
}

func (r *readonlyFileSystem) Rename(oldname, newname string) error {
	return ErrReadOnly
}

func (r *readonlyFileSystem) Remove(path string) error {
	return ErrReadOnly
}

func (r *readonlyFileSystem) RemoveAll(path string) error {
	return ErrReadOnly
}

func (r *readonlyFileSystem) Chmod(path string, perm os.FileMode) error {
	return ErrReadOnly
}

func (r *readonlyFileSystem) Chtimes(path string, atime time.Time, mtime time.Time) error {
	return ErrReadOnly
}

var ErrReadOnly = vfs.ErrReadOnly
