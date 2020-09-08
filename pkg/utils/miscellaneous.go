// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

// MergeMaps takes two maps <a>, <b> and merges them. If <b> defines a value with a key
// already existing in the <a> map, the <a> value for that key will be overwritten.
func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	var values = map[string]interface{}{}

	for i, v := range b {
		existing, ok := a[i]
		values[i] = v

		switch elem := v.(type) {
		case map[string]interface{}:
			if ok {
				if extMap, ok := existing.(map[string]interface{}); ok {
					values[i] = MergeMaps(extMap, elem)
				}
			}
		default:
			values[i] = v
		}
	}

	for i, v := range a {
		if _, ok := values[i]; !ok {
			values[i] = v
		}
	}

	return values
}

// CopyFS copies all files and directories of a filesystem to another.
func CopyFS(src, dst vfs.FileSystem, srcPath, dstPath string) error {
	return vfs.Walk(src, srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dstFilePath := filepath.Join(dstPath, path)
		if info.IsDir() {
			if err := dst.MkdirAll(dstFilePath, info.Mode()); err != nil {
				return err
			}
			return nil
		}

		file, err := src.OpenFile(path, os.O_RDONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()

		dstFile, err := dst.Create(dstFilePath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, file); err != nil {
			return err
		}
		return nil
	})
}
