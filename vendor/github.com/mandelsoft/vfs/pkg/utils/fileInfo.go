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
	"os"
	"time"
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// fileInfo implementing os.FileInfo
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type fileInfo struct {
	fileData FileData
	name     string
}

var _ os.FileInfo = &fileInfo{}

func NewFileInfo(name string, file FileData) os.FileInfo {
	return &fileInfo{name: name, fileData: file}
}

var _ os.FileInfo = &fileInfo{}

func (f *fileInfo) Name() string {
	return f.name
}

func (f *fileInfo) Mode() os.FileMode {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	return f.fileData.Mode()
}

func (f *fileInfo) ModTime() time.Time {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	return f.fileData.ModTime()
}

func (f *fileInfo) IsDir() bool {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	return f.fileData.IsDir()
}

func (f *fileInfo) Sys() interface{} { return nil }

func (f *fileInfo) Size() int64 {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	if f.fileData.IsDir() {
		return int64(42)
	}
	return int64(len(f.fileData.Data()))
}
