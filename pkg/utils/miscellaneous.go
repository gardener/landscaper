// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"
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

// JSONSerializeToGenericObject converts a typed struct to an generic interface using json serialization.
func JSONSerializeToGenericObject(in interface{}) (interface{}, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}

// StringIsOneOf checks whether in is one of s.
func StringIsOneOf(in string, s ...string) bool {
	for _, search := range s {
		if search == in {
			return true
		}
	}
	return false
}

// GetSizeOfDirectory returns the size of all files in a root directory.
func GetSizeOfDirectory(filesystem vfs.FileSystem, root string) (int64, error) {
	var size int64
	err := vfs.Walk(filesystem, root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		size = size + info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return size, nil
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

// YAMLReadFromFile reads a file from a filesystem and decodes it into the given obj
// using YAMl/JSON decoder.
func YAMLReadFromFile(fs vfs.FileSystem, path string, obj interface{}) error {
	data, err := vfs.ReadFile(fs, path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, obj)
}
