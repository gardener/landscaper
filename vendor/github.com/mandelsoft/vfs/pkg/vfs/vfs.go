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

package vfs

import (
	"os"
)

type vfs struct {
	FileSystem
}

func New(fs FileSystem) VFS {
	if v, ok := fs.(VFS); ok {
		return v
	}
	return &vfs{fs}
}

func (fs *vfs) Join(elems ...string) string {
	return Join(fs, elems...)
}

func (fs *vfs) Split(path string) (string, string) {
	return Split(fs, path)
}

func (fs *vfs) Base(path string) string {
	return Base(fs, path)
}

func (fs *vfs) Dir(path string) string {
	return Dir(fs, path)
}

func (fs *vfs) Clean(path string) string {
	return Clean(fs, path)
}

func (fs *vfs) Trim(path string) string {
	return Trim(fs, path)
}

func (fs *vfs) IsAbs(path string) bool {
	return IsAbs(fs, path)
}

func (fs *vfs) IsRoot(path string) bool {
	return IsRoot(fs, path)
}

func (fs *vfs) SplitVolume(path string) (string, string) {
	return SplitVolume(fs, path)
}

func (fs *vfs) SplitPath(path string) (vol string, elems []string, rooted bool) {
	return SplitPath(fs, path)
}

func (fs *vfs) Canonical(path string, exist bool) (string, error) {
	return Canonical(fs, path, exist)
}

func (fs *vfs) Abs(path string) (string, error) {
	return Abs(fs, path)
}

func (fs *vfs) Rel(src, tgt string) (string, error) {
	return Rel(fs, src, tgt)
}

func (fs *vfs) Components(path string) (string, []string) {
	return Components(fs, path)
}

func (fs *vfs) EvalSymlinks(path string) (string, error) {
	return EvalSymlinks(fs, path)
}

func (fs *vfs) Walk(path string, fn WalkFunc) error {
	return Walk(fs, path, fn)
}

func (fs *vfs) Exists(path string) (bool, error) {
	return Exists(fs, path)
}

func (fs *vfs) DirExists(path string) (bool, error) {
	return DirExists(fs, path)
}

func (fs *vfs) FileExists(path string) (bool, error) {
	return FileExists(fs, path)
}

func (fs *vfs) IsDir(path string) (bool, error) {
	return IsDir(fs, path)
}

func (fs *vfs) IsFile(path string) (bool, error) {
	return IsFile(fs, path)
}

func (fs *vfs) ReadFile(path string) ([]byte, error) {
	return ReadFile(fs, path)
}

func (fs *vfs) WriteFile(path string, data []byte, mode os.FileMode) error {
	return WriteFile(fs, path, data, mode)
}

func (fs *vfs) ReadDir(path string) ([]os.FileInfo, error) {
	return ReadDir(fs, path)
}

func (fs *vfs) TempFile(dir, prefix string) (File, error) {
	return TempFile(fs, dir, prefix)
}

func (fs *vfs) TempDir(dir, prefix string) (string, error) {
	return TempDir(fs, dir, prefix)
}

func (fs *vfs) Cleanup() error {
	return Cleanup(fs.FileSystem)
}
