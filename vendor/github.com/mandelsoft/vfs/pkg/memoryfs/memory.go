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
	"time"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type memoryFileSystemAdaper struct{}

func New() vfs.FileSystem {
	adapter := &memoryFileSystemAdaper{}
	return utils.NewFSSupport("MemoryFileSystem", adapter.CreateDir(os.ModePerm), adapter)
}

func (a memoryFileSystemAdaper) CreateFile(perm os.FileMode) utils.FileData {
	return &fileData{mode: os.ModeTemporary | (perm & os.ModePerm), modtime: time.Now()}
}

func (a memoryFileSystemAdaper) CreateDir(perm os.FileMode) utils.FileData {
	return &fileData{mode: os.ModeDir | os.ModeTemporary | (perm & os.ModePerm), entries: DirectoryEntries{}, modtime: time.Now()}
}

func (a memoryFileSystemAdaper) CreateSymlink(link string, perm os.FileMode) utils.FileData {
	return &fileData{mode: os.ModeSymlink | os.ModeTemporary | (perm & os.ModePerm), data: []byte(link), modtime: time.Now()}
}
