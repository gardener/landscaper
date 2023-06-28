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
	"io"
	"os"
	"sort"
	"strings"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type file struct {
	vfs.File
	name  string
	info  *fileData
	files []os.FileInfo
}

func newFileHandle(name string, info *fileData, f vfs.File) vfs.File {
	return &file{f, name, info, nil}
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	var outLength int64

	if f.files == nil {
		list, err := f.File.Readdir(0)
		if err != nil {
			return nil, err
		}
		files := []os.FileInfo{}
		deleted := map[string]struct{}{}
		found := map[string]struct{}{}
		forward := true
		for _, e := range list {
			n := e.Name()
			if n == opaque_del {
				forward = false
			} else {
				if strings.HasPrefix(n, del_prefix) {
					deleted[n[len(del_prefix):]] = struct{}{}
				} else {
					found[n] = struct{}{}
					files = append(files, e)
				}
			}
		}
		if forward {
			file, err := f.info.base.Open(f.info.path)
			if err != nil {
				if !vfs.IsErrNotExist(err) {
					return nil, err
				}
			}
			if file != nil {
				defer file.Close()
				list, err := file.Readdir(0)
				if err != nil {
					return nil, err
				}
				for _, e := range list {
					n := e.Name()
					if n == opaque_del {
						continue
					}
					if _, ok := deleted[n]; ok {
						continue
					}
					if _, ok := found[n]; ok {
						continue
					}
					if strings.HasPrefix(n, del_prefix) {
						continue
					}
					files = append(files, e)
				}
			}
		}

		sort.Sort(utils.FilesSorter(files))
		f.files = files
	}

	var err error
	if count > 0 {
		if len(f.files) < count {
			outLength = int64(len(f.files))
		} else {
			outLength = int64(count)
		}
		if len(f.files) == 0 {
			err = io.EOF
		}
	} else {
		outLength = int64(len(f.files))
	}
	files := f.files[:outLength]
	f.files = f.files[outLength:]
	return files, err
}

func (f *file) Readdirnames(n int) (names []string, err error) {
	fi, err := f.Readdir(n)
	names = make([]string, len(fi))
	for i, f := range fi {
		names[i] = f.Name()
	}
	return names, err
}

func (f *file) Close() error {
	f.files = nil
	return f.File.Close()
}
