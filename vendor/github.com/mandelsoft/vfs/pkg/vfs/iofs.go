/*
 * Copyright 2023 Mandelsoft. All rights reserved.
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
	"io/fs"
	"sort"
)

type iofs struct {
	FileSystem
}

var (
	_ fs.File        = File(nil)
	_ fs.ReadDirFile = File(nil)
)

func (i *iofs) Open(name string) (fs.File, error) {
	return i.FileSystem.Open(name)
}

func (i *iofs) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := i.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dirs, err := f.ReadDir(-1)
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	return dirs, err
}

// AsIoFS maps a virtual filesystem
func AsIoFS(fs FileSystem) fs.ReadDirFS {
	return &iofs{fs}
}
