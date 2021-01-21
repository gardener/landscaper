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

package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type PathMapper interface {
	MapPath(path string) (vfs.FileSystem, string)
}

type MappedFileSystem struct {
	FileSystemBase
	mapper PathMapper
	base   vfs.FileSystem
}

func NewMappedFileSystem(root vfs.FileSystem, mapper PathMapper) *MappedFileSystem {
	return &MappedFileSystem{mapper: mapper, base: root}
}

func (m *MappedFileSystem) Base() vfs.FileSystem {
	return m.base
}

func (m *MappedFileSystem) VolumeName(name string) string {
	return m.base.VolumeName(name)
}

func (m *MappedFileSystem) FSTempDir() string {
	return vfs.PathSeparatorString
}

func (m *MappedFileSystem) Normalize(path string) string {
	return m.base.Normalize(path)
}

func (*MappedFileSystem) Getwd() (string, error) {
	return vfs.PathSeparatorString, nil
}

// isAbs reports whether the path is absolute.
func isAbs(path string) bool {
	return strings.HasPrefix(path, vfs.PathSeparatorString)
}

func (m *MappedFileSystem) mapPath(path string, link ...bool) (vfs.FileSystem, string, string, error) {
	getlink := true
	if len(link) > 0 {
		getlink = link[0]
	}

	r := vfs.PathSeparatorString
	fs, l := m.mapper.MapPath(r)
	links := 0
	path = fs.Normalize(path)

	for path != "" {
		i := 0
		for i < len(path) && vfs.IsPathSeparator(path[i]) {
			i++
		}
		j := i
		for j < len(path) && !vfs.IsPathSeparator(path[j]) {
			j++
		}

		b := path[i:j]
		path = path[j:]

		switch b {
		case ".", "":
			continue
		case "..":
			r, b = vfs.Split(m.base, r)
			if r == "" {
				r = "/"
			}
			fs, l = m.mapper.MapPath(r)
			continue
		}
		fs, l = m.mapper.MapPath(vfs.Join(m.base, r, b))

		fi, err := fs.Lstat(l)
		if vfs.Exists_(err) {
			if err != nil && !os.IsPermission(err) {
				return nil, "", "", err
			}
			if fi.Mode()&os.ModeSymlink != 0 && (getlink || strings.Contains(path, vfs.PathSeparatorString)) {
				links++
				if links > 255 {
					return nil, "", "", errors.New("AbsPath: too many links")
				}
				newpath, err := fs.Readlink(l)
				if err != nil {
					return nil, "", "", err
				}
				newpath = fs.Normalize(newpath)
				vol, newpath := vfs.SplitVolume(m.base, newpath)
				if vol != "" {
					return nil, "", "", fmt.Errorf("volume links not possible: %s: %s", l, vol+newpath)
				}
				if isAbs(newpath) {
					r = "/"
				}
				path = vfs.Join(m.base, newpath, path)
			} else {
				r = vfs.Join(m.base, r, b)
			}
		} else {
			if strings.Contains(path, vfs.PathSeparatorString) {
				return nil, "", "", err
			}
			r = vfs.Join(m.base, r, b)
		}
	}
	return fs, l, r, nil
}

func (m *MappedFileSystem) Chtimes(name string, atime, mtime time.Time) (err error) {
	fs, l, _, err := m.mapPath(name)
	if err != nil {
		return &os.PathError{Op: "chtimes", Path: name, Err: err}
	}
	return fs.Chtimes(l, atime, mtime)
}

func (m *MappedFileSystem) Chmod(name string, mode os.FileMode) (err error) {
	fs, l, _, err := m.mapPath(name)
	if err != nil {
		return &os.PathError{Op: "chmod", Path: name, Err: err}
	}
	return fs.Chmod(l, mode)
}

func (m *MappedFileSystem) Stat(name string) (fi os.FileInfo, err error) {
	fs, l, _, err := m.mapPath(name)
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	return fs.Stat(l)
}

func (m *MappedFileSystem) Rename(oldname, newname string) (err error) {
	oldfs, o, _, err := m.mapPath(oldname, false)
	if err != nil {
		return &os.PathError{Op: "rename", Path: oldname, Err: err}
	}
	newfs, n, _, err := m.mapPath(newname)
	if err != nil {
		return &os.PathError{Op: "rename", Path: newname, Err: err}
	}
	if oldfs == newfs {
		return oldfs.Rename(o, n)
	}
	return fmt.Errorf("no cross filesystem rename operation possible: %s -> %s", oldname, newname)
}

func (m *MappedFileSystem) RemoveAll(name string) (err error) {
	fs, l, _, err := m.mapPath(name, false)
	if err != nil {
		return &os.PathError{Op: "remove_all", Path: name, Err: err}
	}
	return fs.RemoveAll(l)
}

func (m *MappedFileSystem) Remove(name string) (err error) {
	fs, l, _, err := m.mapPath(name, false)
	if err != nil {
		return &os.PathError{Op: "remove", Path: name, Err: err}
	}
	return fs.Remove(l)
}

func (m *MappedFileSystem) OpenFile(name string, flag int, mode os.FileMode) (f vfs.File, err error) {
	fs, l, n, err := m.mapPath(name)
	if err != nil {
		return nil, &os.PathError{Op: "openfile", Path: name, Err: err}
	}
	sourcef, err := fs.OpenFile(l, flag, mode)
	if err != nil {
		return nil, err
	}
	return NewRenamedFile(n, sourcef), nil
}

func (m *MappedFileSystem) Open(name string) (f vfs.File, err error) {
	fs, l, n, err := m.mapPath(name)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}
	sourcef, err := fs.Open(l)
	if err != nil {
		return nil, err
	}
	return NewRenamedFile(n, sourcef), nil
}

func (m *MappedFileSystem) Mkdir(name string, mode os.FileMode) (err error) {
	fs, l, _, err := m.mapPath(name)
	if err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return fs.Mkdir(l, mode)
}

func (m *MappedFileSystem) MkdirAll(name string, mode os.FileMode) (err error) {
	fs, l, _, err := m.mapPath(name)
	if err == nil && fs == m.base {
		return fs.MkdirAll(l, mode)
	}

	_, elems, _ := vfs.SplitPath(m.base, name)

	r := ""
	for _, dir := range elems {
		r = vfs.PathSeparatorString + dir
		fs, l, r, err := m.mapPath(r)
		if err != nil {
			return &os.PathError{Op: "mkdir", Path: name, Err: err}
		}
		fi, err := fs.Stat(l)
		if err == nil {
			if fi.IsDir() {
				continue
			}
			return &os.PathError{Op: "mkdir", Path: name, Err: fmt.Errorf("%s is no dir", r)}
		}
		err = fs.Mkdir(l, mode)
		if err != nil {
			return &os.PathError{Op: "mkdir", Path: name, Err: err}
		}
	}
	return nil
}

func (m *MappedFileSystem) Create(name string) (f vfs.File, err error) {
	fs, l, n, err := m.mapPath(name)
	if err != nil {
		return nil, &os.PathError{Op: "create", Path: name, Err: err}
	}
	sourcef, err := fs.Create(l)
	if err != nil {
		return nil, err
	}
	return NewRenamedFile(n, sourcef), nil
}

func (m *MappedFileSystem) Lstat(name string) (os.FileInfo, error) {
	fs, l, _, err := m.mapPath(name, false)
	if err != nil {
		return nil, &os.PathError{Op: "lstat", Path: name, Err: err}
	}
	return fs.Lstat(l)
}

func (m *MappedFileSystem) Symlink(oldname, newname string) error {
	fs, l, _, err := m.mapPath(newname)
	if err != nil {
		return &os.PathError{Op: "rename", Path: newname, Err: err}
	}
	return fs.Symlink(oldname, l)
}

func (m *MappedFileSystem) Readlink(name string) (string, error) {
	fs, l, _, err := m.mapPath(name, false)
	if err != nil {
		return "", &os.PathError{Op: "readlink", Path: name, Err: err}
	}
	return fs.Readlink(l)
}
