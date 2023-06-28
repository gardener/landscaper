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

package layerfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type LayerFileSystem struct {
	utils.FileSystemBase
	layer vfs.FileSystem
	base  vfs.FileSystem
}

var _ vfs.FileSystemCleanup = (*LayerFileSystem)(nil)

func New(layer, base vfs.FileSystem) vfs.FileSystem {
	fs := &LayerFileSystem{layer: layer, base: base}
	return fs
}

func (l *LayerFileSystem) Cleanup() error {
	err := vfs.Cleanup(l.layer)
	err2 := vfs.Cleanup(l.base)

	if err == nil {
		if err2 != nil {
			return err2
		}
	} else {
		if err2 != nil {
			return fmt.Errorf("error cleaning layer: layer %s, base %s", err2.Error(), err.Error())
		}
	}
	return err
}

func (l *LayerFileSystem) Name() string {
	return fmt.Sprintf("LayerFileSystem %s[%s]", l.layer, l.base)
}

func (l *LayerFileSystem) findFile(name string, link ...bool) (*fileData, string, error) {
	_, _, f, n, err := l.createInfo(name, link...)
	if err != nil {
		return nil, n, err
	}
	if f == nil {
		err = vfs.ErrNotExist
	}
	return f, n, err
}

func (l *LayerFileSystem) createInfo(name string, link ...bool) (*fileData, string, *fileData, string, error) {
	d, dn, f, fn, err := utils.EvaluatePath(l, newFileData(l.layer, l.base, vfs.PathSeparatorString, nil), name, link...)
	return asFileData(d), dn, asFileData(f), fn, err
}

func (l *LayerFileSystem) propagateDirectories(path string) error {
	_, elems, _ := vfs.SplitPath(l.layer, path)

	path = "/"
	for _, e := range elems {
		path = vfs.Join(l.layer, path, e)
		fi, err := l.layer.Lstat(path)
		if err == nil {
			continue
		}
		if !vfs.IsErrNotExist(err) {
			return err
		}
		fi, err = l.base.Lstat(path)
		if err != nil {
			return err
		}
		err = l.layer.Mkdir(path, fi.Mode())
		if err != nil {
			return err
		}
		err = l.layer.Chtimes(path, time.Now(), fi.ModTime())
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *LayerFileSystem) copy(fi os.FileInfo, path string) (*fileData, error) {
	f := newFileData(l.layer, l.base, path, fi)
	if fi.IsDir() {
		return f, l.propagateDirectories(path)
	}

	dir := vfs.Dir(l.layer, path)
	err := l.propagateDirectories(dir)
	if err != nil {
		return f, err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		old, err := f.fs.Readlink(f.path)
		if err != nil {
			return f, err
		}
		return f, l.layer.Symlink(old, f.path)
	}
	if !fi.Mode().IsRegular() {
		return f, errors.New("file type not supported")
	}
	return f, vfs.CopyFile(l.base, path, l.layer, path)
}

func (l *LayerFileSystem) create(name string, fn func(path string, deleted bool) (vfs.File, error)) (vfs.File, error) {
	parent, _, f, n, err := l.createInfo(name)
	if err != nil {
		return nil, err
	}
	if f != nil {
		return nil, vfs.ErrExist
	}

	path := vfs.Join(l.layer, parent.path, n)
	// entry was formerly deleted from layer, if
	// - it is explicitly marked as deleted
	// - the complete base folder content is marked as deleted AND the base layer contains the entry
	del := vfs.Join(l.layer, parent.path, del_prefix+n)
	deleted, _ := vfs.Exists(l.layer, del)
	if !deleted {
		deleted, _ = vfs.Exists(l.layer, vfs.Join(l.layer, parent.path, opaque_del))
		if deleted {
			deleted, _ = vfs.Exists(l.base, path)
		}
	}

	err = l.propagateDirectories(parent.path)
	if err != nil {
		return nil, err
	}
	file, err := fn(path, deleted)
	if err == nil {
		if deleted {
			err := l.layer.Remove(del)
			if err != nil && !vfs.IsErrNotExist(err) {
				l.layer.Remove(path)
				return nil, err
			}
		}
	}
	return file, err
}

func (l *LayerFileSystem) Create(name string) (vfs.File, error) {
	return l.create(name, func(path string, deleted bool) (vfs.File, error) { return l.layer.Create(path) })
}

func (l *LayerFileSystem) Mkdir(name string, perm os.FileMode) error {
	_, err := l.create(name, func(path string, deleted bool) (vfs.File, error) {
		err := l.layer.Mkdir(path, perm)
		if err == nil && deleted {
			file, err := l.layer.Create(vfs.Join(l.layer, path, opaque_del))
			if err == nil {
				file.Close()
			} else {
				l.layer.Remove(path)
				// TODO error handling
			}
			return nil, err
		}
		return nil, err
	})
	return err
}

func (l *LayerFileSystem) MkdirAll(path string, perm os.FileMode) error {
	_, elems, _ := vfs.SplitPath(l, path)
	cur := "/"
	for _, e := range elems {
		cur = vfs.Join(l, cur, e)
		if ok, err := vfs.Exists(l, cur); !ok || err != nil {
			err := l.Mkdir(cur, perm)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *LayerFileSystem) Open(name string) (vfs.File, error) {
	f, n, err := l.findFile(name)
	if err != nil {
		return nil, err
	}
	file, err := f.fs.Open(f.path)
	if err != nil {
		return nil, err
	}
	if f.base == nil {
		return file, err
	}
	return newFileHandle(n, f, file), nil
}

func (l *LayerFileSystem) OpenFile(name string, flags int, perm os.FileMode) (vfs.File, error) {
	d, _, f, n, err := l.createInfo(name)
	if err != nil {
		return nil, err
	}
	var file vfs.File
	if f == nil {
		if flags&(os.O_CREATE) == 0 {
			return nil, vfs.NewPathError("create", name, os.ErrNotExist)
		}
		return l.create(name, func(path string, deleted bool) (vfs.File, error) {
			return l.layer.OpenFile(path, flags, perm)
		})
	}

	if f.base == nil {
		fi, err := f.fs.Lstat(f.path)
		if err != nil {
			return nil, err
		}
		if fi.Mode().IsRegular() {
			if flags&(os.O_WRONLY|os.O_RDWR) != 0 {
				if (flags & os.O_TRUNC) != 0 {
					err = l.propagateDirectories(d.path)
					if err != nil {
						return nil, err
					}
					return l.layer.OpenFile(vfs.Join(l.layer, d.path, n), flags|os.O_CREATE, perm)
				}
				f, err = l.copy(fi, f.path)
				if err != nil {
					return nil, err
				}
				return f.fs.OpenFile(f.path, flags, perm)
			}
		}
	}
	file, err = f.fs.OpenFile(f.path, flags, perm)
	if err != nil {
		return nil, err
	}
	if f.base == nil {
		return file, err
	}
	return newFileHandle(n, f, file), nil
}

func markerFor(path string) string {
	i := strings.LastIndex(path, vfs.PathSeparatorString)
	return path[:i+1] + del_prefix + path[i+1:]
}

func (l *LayerFileSystem) Remove(name string) error {
	d, _, f, n, err := l.createInfo(name, false)
	if err != nil {
		return err
	}

	if f == nil {
		return vfs.ErrNotExist
	}
	fi, err := f.Lstat()
	if err != nil {
		return err
	}
	if fi.IsDir() {
		file, err := f.fs.Open(f.path)
		if err != nil {
			return err
		}
		defer file.Close()
		names, err := file.Readdirnames(1)
		if err != nil && err != io.EOF {
			return err
		}
		if len(names) > 0 {
			return vfs.NewPathError("remove", name, vfs.ErrNotEmpty)
		}
	}
	if n == "" {
		return errors.New("cannot delete root dir")
	}
	if f.base == nil {
		if fi.IsDir() {
			err = l.propagateDirectories(f.path)
			if err != nil {
				return err
			}

			err = vfs.Touch(l.layer, vfs.Join(l.layer, f.path, opaque_del), os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			err = l.propagateDirectories(d.path)
			if err != nil {
				return err
			}
		}
	} else {
		err = f.fs.Remove(f.path)
		if err != nil {
			return err
		}
	}
	err = vfs.Touch(l.layer, markerFor(f.path), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (l *LayerFileSystem) RemoveAll(name string) error {
	d, _, f, n, err := l.createInfo(name, false)
	if err != nil {
		return err
	}
	if n == "" {
		return errors.New("cannot delete root dir")
	}

	if f.base != nil {
		err = l.layer.RemoveAll(f.path)
		if err != nil {
			return err
		}

		_, err := f.base.Lstat(f.path)
		if err == nil || !vfs.IsErrNotExist(err) {
			return err
		}
	}

	err = l.propagateDirectories(d.path)
	if err != nil {
		return err
	}
	marker := markerFor(f.path)
	err = vfs.Touch(l.layer, marker, os.ModePerm)
	return err
}

func (l *LayerFileSystem) Rename(oldname, newname string) error {
	odir, _, fo, o, err := l.createInfo(oldname, false)
	if err != nil {
		return err
	}
	if o == "" {
		return errors.New("cannot rename root dir")
	}
	ndir, _, fn, _, err := l.createInfo(newname)
	if err != nil {
		return err
	}
	if fo == nil {
		return os.ErrNotExist
	}
	if fn != nil {
		return os.ErrExist
	}
	// TODO rename
	_, _ = ndir, odir
	return fmt.Errorf("rename not implemented yet")
}

func (l *LayerFileSystem) Lstat(name string) (os.FileInfo, error) {
	f, _, err := l.findFile(name, false)
	if err != nil {
		return nil, err
	}
	return f.fs.Lstat(f.path)
}

func (l *LayerFileSystem) Stat(name string) (os.FileInfo, error) {
	f, _, err := l.findFile(name)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, os.ErrNotExist
	}
	return f.fs.Stat(f.path)
}

func (l *LayerFileSystem) Chmod(name string, mode os.FileMode) error {
	f, _, err := l.findFile(name)
	if err != nil {
		return err
	}
	if f.base == nil {
		fi, err := f.fs.Lstat(f.path)
		if err != nil {
			return err
		}
		if fi.Mode().Perm() == mode.Perm() {
			return nil
		}
		f, err = l.copy(fi, f.path)
		if err != nil {
			return err
		}
	}
	f.fs.Chmod(f.path, mode)
	return nil
}

func (l *LayerFileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	f, _, err := l.findFile(name)
	if err != nil {
		return err
	}

	if f.base == nil {
		fi, err := f.fs.Lstat(f.path)
		if err != nil {
			return err
		}
		if fi.ModTime() == mtime {
			return nil
		}
		f, err = l.copy(fi, f.path)
		if err != nil {
			return err
		}
	}

	f.fs.Chtimes(f.path, atime, mtime)
	return nil
}

func (l *LayerFileSystem) Symlink(oldname, newname string) error {
	_, err := l.create(newname, func(path string, deleted bool) (vfs.File, error) {
		return nil, l.layer.Symlink(oldname, path)
	})
	return err
}

func (l *LayerFileSystem) Readlink(name string) (string, error) {
	f, _, err := l.findFile(name, false)
	if err != nil {
		return "", err
	}
	if f.IsSymlink() {
		return f.GetSymlink(), nil
	}
	return "", vfs.NewPathError("readlink", name, errors.New("no symlink"))
}
