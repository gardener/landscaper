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

package yamlfs

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type yamlFileSystemAdaper struct{}

type YamlFileSystem struct {
	vfs.FileSystem
	data map[interface{}]interface{}
}

func New(data []byte) (*YamlFileSystem, error) {
	m := map[interface{}]interface{}{}
	err := yaml.Unmarshal(data, m)
	if err != nil {
		return nil, err
	}
	return NewByData(m), nil
}

func NewByPath(fs vfs.FileSystem, path string) (*YamlFileSystem, error) {
	var data []byte
	var err error

	if fs == nil {
		data, err = ioutil.ReadFile(path)
	} else {
		data, err = vfs.ReadFile(fs, path)
	}
	if err != nil {
		return nil, err
	}
	return New(data)
}

func NewByData(data map[interface{}]interface{}) *YamlFileSystem {
	adapter := &yamlFileSystemAdaper{}
	if data == nil {
		data = map[interface{}]interface{}{}
	}
	return &YamlFileSystem{utils.NewFSSupport("YamlFileSystem", newFileDirData(data), adapter), data}
}

func (y *YamlFileSystem) Data() ([]byte, error) {
	return yaml.Marshal(y.data)
}

//////////////////////////////////////////////////////////////////////////////

func (a yamlFileSystemAdaper) CreateFile(perm os.FileMode) utils.FileData {
	return newFileDataData(nil, "", []byte{}, FromBinary)
}

func (a yamlFileSystemAdaper) CreateDir(perm os.FileMode) utils.FileData {
	return newFileDirData(map[interface{}]interface{}{})
}

func (a yamlFileSystemAdaper) CreateSymlink(link string, perm os.FileMode) utils.FileData {
	m := map[interface{}]interface{}{}
	m[typeField] = typeSYMLINK
	m[valueField] = link
	return newFileSymlinkData(m)
}
