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

package memoryfs

import (
	"os"
	"sync"
	"time"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type fileData struct {
	sync.Mutex
	data    []byte
	entries DirectoryEntries
	mode    os.FileMode
	modtime time.Time
}

var _ utils.FileData = &fileData{}

func (f *fileData) Data() []byte {
	return f.data
}

func (f *fileData) SetData(data []byte) {
	f.data = data
}

func (f *fileData) Files() []os.FileInfo {
	return f.entries.Files()
}

func (f *fileData) IsDir() bool {
	return f.mode&os.ModeType == os.ModeDir
}

func (f *fileData) IsFile() bool {
	return f.mode&os.ModeType == 0
}

func (f *fileData) IsSymlink() bool {
	return (f.mode & os.ModeType) == os.ModeSymlink
}

func (f *fileData) GetSymlink() string {
	if f.IsSymlink() {
		return string(f.data)
	}
	return ""
}

func (f *fileData) Mode() os.FileMode {
	return f.mode
}

func (f *fileData) SetMode(mode os.FileMode) {
	f.mode = mode
}

func (f *fileData) ModTime() time.Time {
	return f.modtime
}

func (f *fileData) SetModTime(mtime time.Time) {
	f.modtime = mtime
}

func (f *fileData) GetEntry(name string) (utils.FileDataDirAccess, error) {
	if !f.IsDir() {
		return nil, vfs.ErrNotDir
	}
	e, ok := f.entries[name]
	if ok {
		return e, nil
	}
	return nil, vfs.ErrNotExist
}

func (f *fileData) Add(name string, s utils.FileData) error {
	if !f.IsDir() {
		return vfs.ErrNotDir
	}
	if _, ok := f.entries[name]; ok {
		return os.ErrExist
	}
	f.entries.Add(name, s.(*fileData))
	f.SetModTime(time.Now())
	return nil
}

func (f *fileData) Del(name string) error {
	if !f.IsDir() {
		return vfs.ErrNotDir
	}
	_, ok := f.entries[name]
	if !ok {
		return vfs.ErrNotExist
	}
	delete(f.entries, name)
	return nil
}
