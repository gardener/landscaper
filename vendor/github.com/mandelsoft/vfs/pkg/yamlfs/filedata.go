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

package yamlfs

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

var modtime = time.Now()

const typeField = "$type"
const valueField = "value"
const binaryField = "binary"

const typeDIR = "directory"
const typeSYMLINK = "symlink"

const typeYAML = "yaml"
const typeJSON = "json"

var UseStandardYAMLBinary = true

func validType(t string) bool {
	switch t {
	case typeDIR, typeSYMLINK, typeYAML, typeJSON:
		return true
	default:
		return false
	}
}

type DataMarshaller func(data []byte) interface{}

type source interface {
	Source() interface{}
}

type fileBaseData struct {
	sync.Mutex
	mode    os.FileMode
	modtime time.Time
	data    map[interface{}]interface{}
}

var _ utils.FileData = &fileBaseData{}

func newFileBaseData(t os.FileMode, data map[interface{}]interface{}) fileBaseData {
	return fileBaseData{mode: os.ModePerm | (t & os.ModeType) | os.ModeTemporary, modtime: modtime, data: data}
}

func (f *fileBaseData) Source() interface{} {
	return f.data
}

func (f *fileBaseData) Data() []byte {
	return nil
}

func (f *fileBaseData) SetData(data []byte) {
}

func (f *fileBaseData) Files() []os.FileInfo {
	return nil
}

func (f *fileBaseData) IsDir() bool {
	return f.mode&os.ModeType == os.ModeDir
}

func (f *fileBaseData) IsFile() bool {
	return f.mode&os.ModeType == 0
}

func (f *fileBaseData) IsSymlink() bool {
	return (f.mode & os.ModeType) == os.ModeSymlink
}

func (f *fileBaseData) GetSymlink() string {
	return ""
}

func (f *fileBaseData) Mode() os.FileMode {
	return f.mode
}

func (f *fileBaseData) SetMode(mode os.FileMode) {
	f.mode = mode
}

func (f *fileBaseData) ModTime() time.Time {
	return f.modtime
}

func (f *fileBaseData) SetModTime(mtime time.Time) {
	f.modtime = mtime
}

func (f *fileBaseData) GetEntry(name string) (utils.FileDataDirAccess, error) {
	return nil, vfs.ErrNotDir
}

func (f *fileBaseData) Add(name string, s utils.FileData) error {
	return vfs.ErrNotDir
}

func (f *fileBaseData) Del(name string) error {
	return vfs.ErrNotDir
}

///////////////////////////////////////////////////////////////////////////////////////////

func convertMap(m map[string]interface{}) map[interface{}]interface{} {
	r := map[interface{}]interface{}{}
	for k, v := range m {
		r[k] = v
	}
	return r
}

func _convert(v interface{}) (interface{}, error) {
	var r interface{}
	var err error
	if v == nil {
		return nil, nil
	}
	switch e := v.(type) {
	case []interface{}:
		a := []interface{}{}
		for _, v := range e {
			v, err = _convert(v)
			if err != nil {
				return nil, err
			}
			a = append(a, v)
		}
		r = a
	case map[string]interface{}:
		m := map[string]interface{}{}
		for n, v := range e {
			m[n], err = _convert(v)
			if err != nil {
				return nil, err
			}
		}
		r = m
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range e {
			if n, ok := k.(string); ok {
				m[n], err = _convert(v)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, errors.New("invalid key")
			}
		}
		r = m
	default:
		r = e
	}
	return r, nil
}

func convertJson(v interface{}) ([]byte, error) {
	c, err := _convert(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(c)
}

type fileDirData struct {
	fileBaseData
	entries map[string]utils.FileData
}

var _ utils.FileData = &fileDirData{}

func newFileDirData(m map[interface{}]interface{}) utils.FileData {
	return &fileDirData{fileBaseData: newFileBaseData(os.ModeDir, m)}
}

func (f *fileDirData) prepare() {
	if f.entries == nil {
		f.entries = map[string]utils.FileData{}
		for k, e := range f.data {
			n, ok := k.(string)
			if !ok {
				continue
			}
			var fd utils.FileData
			switch entry := e.(type) {
			case []interface{}:
				fd = nil
			case map[interface{}]interface{}:
				{
					if _, ok := entry["$type"]; ok {
						fd = newFileMapData(entry)
					} else {
						fd = newFileDirData(entry)
					}
				}
			case map[string]interface{}:
				m := convertMap(entry)
				if _, ok := entry["$type"]; ok {
					fd = newFileMapData(m)
				} else {
					fd = newFileDirData(m)
				}
			default:
				fd = newFileDataData(f.data, n, ToBinary(e), FromBinary)
			}
			if fd != nil {
				f.entries[n] = fd
			}
		}
	}
}

func (f *fileDirData) Files() []os.FileInfo {
	files := []os.FileInfo{}

	f.prepare()
	for n, e := range f.entries {
		files = append(files, utils.NewFileInfo(n, e))
	}
	return files
}

func (f *fileDirData) GetEntry(name string) (utils.FileDataDirAccess, error) {
	f.prepare()
	e, ok := f.entries[name]
	if ok {
		return e, nil
	}
	return nil, vfs.ErrNotExist
}

func (f *fileDirData) Add(name string, s utils.FileData) error {
	if _, ok := f.entries[name]; ok {
		return os.ErrExist
	}
	if d, ok := s.(*fileDataData); ok {
		if d.field == "" {
			d.field = name
			d.data = f.data
		} else {
			f.data[name] = s.(source).Source()
		}
	} else {
		f.data[name] = s.(source).Source()
	}
	f.entries[name] = s
	f.SetModTime(time.Now())
	return nil
}

func (f fileDirData) Del(name string) error {
	_, ok := f.entries[name]
	if !ok {
		return vfs.ErrNotExist
	}
	delete(f.data, name)
	delete(f.entries, name)
	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////

type fileSymlinkData struct {
	fileBaseData
}

var _ utils.FileData = &fileSymlinkData{}

func newFileSymlinkData(m map[interface{}]interface{}) utils.FileData {
	return &fileSymlinkData{
		fileBaseData: newFileBaseData(os.ModeSymlink, m),
	}
}

func (f *fileSymlinkData) GetSymlink() string {
	return f.data[valueField].(string)
}

///////////////////////////////////////////////////////////////////////////////////////////

const BinaryStartMarker = "---Start Binary---"
const BinaryEndMarker = "---End Binary---"

func ToBinary(data interface{}) []byte {
	if data == nil {
		return []byte{}
	}
	switch e := data.(type) {
	case string:
		lines := strings.Split(e, "\n")
		for i := 0; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "" {
				lines = append(lines[:i], lines[i+1:]...)
				i--
			}
		}
		if len(lines) > 2 {
			if lines[0] == BinaryStartMarker && lines[len(lines)-1] == BinaryEndMarker {
				s := strings.Join(lines[1:len(lines)-1], "")
				d, err := base64.StdEncoding.DecodeString(s)
				if err != nil {
					return []byte(err.Error())
				}
				return d
			}
		}
		return []byte(e)
	case map[string]interface{}, []interface{}:
		d, err := yaml.Marshal(e)
		if err != nil {
			return []byte(err.Error())
		}
		return d
	case []byte:
		return e
	default:
		return []byte(fmt.Sprintf("%s", e))
	}
}

func FromBinary(data []byte) interface{} {
	if UseStandardYAMLBinary {
		return string(data)
	}
	b := base64.StdEncoding.EncodeToString(data)
	n := "\n"
	for len(b) > 120 {
		n = n + b[:120] + "\n"
		b = b[120:]
	}
	if len(b) > 0 {
		n = n + b + "\n"
	}
	return BinaryStartMarker + n + BinaryEndMarker
}

type fileDataData struct {
	fileBaseData
	field      string
	content    []byte
	marshaller DataMarshaller
}

var _ utils.FileData = &fileDataData{}

func newFileDataData(m map[interface{}]interface{}, n string, c []byte, marshaller DataMarshaller) utils.FileData {
	return &fileDataData{
		fileBaseData: newFileBaseData(0, m),
		field:        n,
		content:      c,
		marshaller:   marshaller,
	}
}

func (f *fileDataData) Data() []byte {
	return f.content
}

func (f *fileDataData) SetData(data []byte) {
	f.content = data
	if data != nil && len(data) == 0 {
		f.data[f.field] = nil
	} else {
		v := f.marshaller(data)
		if v != nil {
			f.data[f.field] = v
		}
	}
}

///////////////////////////////////////////////////////////////////////////////////////////

func unmarshalYAML(data []byte) interface{} {
	result := map[interface{}]interface{}{}
	err := yaml.Unmarshal(data, result)
	if err != nil {
		return nil
	}
	return result
}

func unmarshalJSON(data []byte) interface{} {
	result := map[interface{}]interface{}{}
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil
	}
	return result
}

func newFileMapData(m map[interface{}]interface{}) utils.FileData {
	v := m[valueField]
	bin := m[binaryField]
	switch m[typeField] {
	case typeDIR:
		if v == nil {
			v = map[string]interface{}{}
			m[valueField] = v
		}
		if m, ok := v.(map[interface{}]interface{}); ok {
			return newFileDirData(m)
		}
		if m, ok := v.(map[string]interface{}); ok {
			return newFileDirData(convertMap(m))
		}
		return nil
	case typeSYMLINK:
		if v == nil {
			return nil
		}
		if _, ok := v.(string); ok {
			return newFileSymlinkData(m)
		}
		return nil
	case typeYAML:
		b, err := yaml.Marshal(v)
		if err != nil {
			return nil
		}
		return newFileDataData(m, valueField, b, unmarshalYAML)
	case typeJSON:
		b, err := convertJson(v)
		if err != nil {
			return nil
		}
		return newFileDataData(m, valueField, b, unmarshalJSON)
	default:
		if v != nil {
			return newFileDataData(m, valueField, ToBinary(v), FromBinary)
		}
		return newFileDataData(m, binaryField, ToBinary(bin), FromBinary)
	}
}

///////////////////////////////////////////////////////////////////////////////////////////
