// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

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

// StringIsOneOf checks whether in is one of s.
func StringIsOneOf(in string, s ...string) bool {
	for _, search := range s {
		if search == in {
			return true
		}
	}
	return false
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
