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
	"os"
	"strings"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

const opaque_del = ".wh..wh..opq"
const del_prefix = ".wh."

type fileData struct {
	fs   vfs.FileSystem
	base vfs.FileSystem
	path string
	fi   os.FileInfo
}

func asFileData(data utils.FileDataDirAccess) *fileData {
	if data == nil {
		return nil
	}
	return data.(*fileData)
}

func newFileData(fs vfs.FileSystem, base vfs.FileSystem, path string, fi os.FileInfo) *fileData {
	return &fileData{fs: fs, base: base, path: path, fi: fi}
}

func (f *fileData) Lock() {
}

func (f *fileData) Unlock() {
}

func (f *fileData) GetEntry(name string) (utils.FileDataDirAccess, error) {
	if strings.HasPrefix(name, del_prefix) || name == opaque_del {
		return nil, vfs.ErrNotExist
	}
	file, err := f.fs.Open(f.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	names, err := file.Readdirnames(0)
	if err != nil {
		return nil, vfs.ErrNotDir
	}

	del := del_prefix + name
	fpath := vfs.Join(f.fs, f.path, name)
	forward := f.base != nil
	for _, n := range names {
		switch n {
		case name:
			fi, err := f.fs.Lstat(fpath)
			if err != nil {
				return nil, err
			}
			return newFileData(f.fs, f.base, fpath, fi), nil
		case del:
			return nil, vfs.ErrNotExist
		case opaque_del:
			forward = false
		}
	}
	if forward {
		fi, err := f.base.Lstat(fpath)
		if err != nil {
			return nil, err
		}
		return newFileData(f.base, nil, fpath, fi), nil
	}
	return nil, vfs.ErrNotExist
}

func (f *fileData) GetSymlink() string {
	l, err := f.fs.Readlink(f.path)
	if err != nil {
		return ""
	}
	return l
}

func (f *fileData) IsSymlink() bool {
	fi, err := f.Lstat()
	if err != nil {
		return false
	}
	return fi.Mode()&(os.ModeType) == os.ModeSymlink
}

func (f *fileData) IsDir() bool {
	fi, err := f.Lstat()
	if err != nil {
		return false
	}
	return fi.Mode()&(os.ModeType) == os.ModeDir
}

func (f *fileData) Lstat() (os.FileInfo, error) {
	if f.fi == nil {
		fi, err := f.fs.Lstat(f.path)
		if err != nil {
			return nil, err
		}
		f.fi = fi
	}
	return f.fi, nil
}
