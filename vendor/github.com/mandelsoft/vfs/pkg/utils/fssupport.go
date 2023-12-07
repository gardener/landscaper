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

package utils

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type SupportAdapter interface {
	CreateFile(perm os.FileMode) FileData
	CreateDir(perm os.FileMode) FileData
	CreateSymlink(oldname string, perm os.FileMode) FileData
}

type FileSystemSupport struct {
	FileSystemBase
	name    string
	root    FileData
	adapter SupportAdapter
}

func NewFSSupport(name string, root FileData, adapter SupportAdapter) vfs.FileSystem {
	return &FileSystemSupport{name: name, root: root, adapter: adapter}
}

func (m *FileSystemSupport) Name() string {
	return m.name
}

func (m *FileSystemSupport) findFile(name string, link ...bool) (FileData, string, error) {
	_, _, f, n, err := m.createInfo(name, link...)
	if err != nil {
		return nil, n, err
	}
	if f == nil {
		err = os.ErrNotExist
	}
	return f, n, err
}

func asFileData(a FileDataDirAccess) FileData {
	if a == nil {
		return nil
	}
	return a.(FileData)
}

func (m *FileSystemSupport) createInfo(name string, link ...bool) (FileData, string, FileData, string, error) {
	d, dn, f, fn, err := EvaluatePath(m, m.root, name, link...)
	return asFileData(d), dn, asFileData(f), fn, err
}

func (m *FileSystemSupport) Create(name string) (vfs.File, error) {
	parent, _, f, n, err := m.createInfo(name)
	if err != nil {
		return nil, err
	}
	if f != nil {
		if f.Mode()&fs.ModeType != 0 {
			return nil, fs.ErrExist
		}
		h := newFileHandle(n, f)
		err := h.Truncate(0)
		if err != nil {
			return nil, err
		}
		return h, nil
	}

	f = m.adapter.CreateFile(os.ModePerm)
	parent.Lock()
	defer parent.Unlock()
	err = parent.Add(n, f)
	if err != nil {
		return nil, err
	}
	return newFileHandle(n, f), nil
}

func (m *FileSystemSupport) Mkdir(name string, perm os.FileMode) error {
	parent, _, f, n, err := m.createInfo(name)
	if err != nil {
		return err
	}
	if f != nil {
		return os.ErrExist
	}
	parent.Lock()
	defer parent.Unlock()
	return parent.Add(n, m.adapter.CreateDir(perm))
}

func (m *FileSystemSupport) MkdirAll(path string, perm os.FileMode) error {
	path, err := vfs.Canonical(m, path, false)
	if err != nil {
		return err
	}
	_, elems, _ := vfs.SplitPath(m, path)
	parent := m.root
	for i, e := range elems {
		parent.Lock()
		next, err := parent.GetEntry(e)
		if err != nil && err != os.ErrNotExist {
			parent.Unlock()
			return &os.PathError{Op: "mkdirall", Path: strings.Join(elems[:i+1], vfs.PathSeparatorString), Err: err}
		}
		if next == nil {
			next = m.adapter.CreateDir(perm)
			parent.Add(e, next.(FileData))
		}
		parent.Unlock()
		parent = next.(FileData)
	}
	return nil
}

func (m *FileSystemSupport) Open(name string) (vfs.File, error) {
	f, _, err := m.findFile(name)
	if err != nil {
		return nil, err
	}
	return newFileHandle(name, f), nil
}

func (m *FileSystemSupport) OpenFile(name string, flags int, perm os.FileMode) (vfs.File, error) {
	dir, _, f, n, err := m.createInfo(name)
	if err != nil {
		return nil, err
	}
	if f == nil {
		if flags&(os.O_CREATE) == 0 {
			return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
		}
		f = m.adapter.CreateFile(perm)
		dir.Lock()
		err = dir.Add(n, f)
		a, _ := dir.GetEntry(n)
		if err != nil {
			if !vfs.IsErrExist(err) {
				dir.Unlock()
				return nil, &os.PathError{Op: "open", Path: name, Err: err}
			}
			if flags&os.O_EXCL != 0 {
				return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrExist}
			}
			f = a.(FileData)
		}
		dir.Unlock()
	} else {
		if flags&os.O_EXCL != 0 {
			return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrExist}
		}
	}
	h := newFileHandle(name, f)

	if flags&(os.O_RDONLY|os.O_WRONLY|os.O_RDWR) == os.O_RDONLY {
		h.readOnly = true
	} else {
		if flags&os.O_APPEND != 0 {
			_, err = h.Seek(0, os.SEEK_END)
		}
		if err == nil && flags&os.O_TRUNC > 0 && flags&(os.O_RDWR|os.O_WRONLY) > 0 {
			err = h.Truncate(0)
		}
		if err != nil {
			h.Close()
			return nil, err
		}
	}
	return h, nil
}

func (m *FileSystemSupport) Remove(name string) error {
	dir, _, f, n, err := m.createInfo(name, false)
	if err != nil {
		return err
	}

	if f == nil {
		return os.ErrNotExist
	}
	f.Lock()
	defer f.Unlock()
	if f.IsDir() {
		if len(f.Files()) > 0 {
			return &os.PathError{Op: "remove", Path: name, Err: ErrNotEmpty}
		}
	}
	if n == "" {
		return errors.New("cannot delete root dir")
	}
	dir.Lock()
	defer dir.Unlock()
	return dir.Del(n)
}

func (m *FileSystemSupport) RemoveAll(name string) error {
	dir, _, _, n, err := m.createInfo(name, false)
	if err != nil {
		return err
	}
	if n == "" {
		return errors.New("cannot delete root dir")
	}
	dir.Lock()
	defer dir.Unlock()
	return dir.Del(n)
}

func (m *FileSystemSupport) Rename(oldname, newname string) error {
	odir, _, fo, o, err := m.createInfo(oldname, false)
	if err != nil {
		return err
	}
	if o == "" {
		return errors.New("cannot rename root dir")
	}
	ndir, _, fn, n, err := m.createInfo(newname)
	if err != nil {
		return err
	}
	if fo == nil {
		return os.ErrNotExist
	}
	if fn != nil {
		return os.ErrExist
	}

	ndir.Lock()
	err = ndir.Add(n, fo)
	ndir.Unlock()
	if err == nil {
		odir.Lock()
		odir.Del(o)
		odir.Unlock()
	}
	return err
}

func (m *FileSystemSupport) Lstat(name string) (os.FileInfo, error) {
	f, n, err := m.findFile(name, false)
	if err != nil {
		return nil, err
	}
	return NewFileInfo(n, f), nil
}

func (m *FileSystemSupport) Stat(name string) (os.FileInfo, error) {
	f, n, err := m.findFile(name)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, os.ErrNotExist
	}
	return NewFileInfo(n, f), nil
}

func (m *FileSystemSupport) Chmod(name string, mode os.FileMode) error {
	f, _, err := m.findFile(name)
	if err != nil {
		return err
	}
	f.Lock()
	defer f.Unlock()
	f.SetMode((f.Mode() & (^os.ModePerm)) | (mode & os.ModePerm))
	return nil
}

func (m *FileSystemSupport) Chtimes(name string, atime time.Time, mtime time.Time) error {
	f, _, err := m.findFile(name)
	if err != nil {
		return err
	}
	f.Lock()
	defer f.Unlock()
	f.SetModTime(mtime)
	return nil
}

func (m *FileSystemSupport) Symlink(oldname, newname string) error {
	parent, _, _, n, err := m.createInfo(newname)
	if err != nil {
		return err
	}
	parent.Lock()
	defer parent.Unlock()
	return parent.Add(n, m.adapter.CreateSymlink(oldname, os.ModePerm))
}

func (m *FileSystemSupport) Readlink(name string) (string, error) {
	f, _, err := m.findFile(name, false)
	if err != nil {
		return "", err
	}
	f.Lock()
	defer f.Unlock()
	if f.IsSymlink() {
		return f.GetSymlink(), nil
	}
	return "", &os.PathError{Op: "readlink", Path: name, Err: errors.New("no symlink")}
}
