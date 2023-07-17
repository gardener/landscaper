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

package vfs

import (
	"os"
	"sort"

	"github.com/mandelsoft/filepath/pkg/filepath"
)

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
// adapted from https://golang.org/src/path/filepath/path.go
func readDirNames(fs FileSystem, dirname string) ([]string, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// adapted from https://golang.org/src/path/filepath/path.go
func walkFS(fs FileSystem, path string, info os.FileInfo, err error, walkFn WalkFunc) error {
	err1 := walkFn(path, info, err)
	if err != nil || err1 != nil {
		if err1 == SkipDir {
			return nil
		}
		return err1
	}

	if info == nil || !info.IsDir() {
		return nil
	}

	names, err := readDirNames(fs, path)
	if err != nil {
		err := walkFn(path, info, err)
		if err == SkipDir {
			return nil
		}
		return err
	}

	for _, name := range names {
		filename := Join(fs, path, name)
		fileInfo, err := fs.Lstat(filename)

		err = walkFS(fs, filename, fileInfo, err, walkFn)
		if err != nil {
			if err == SkipDir {
				return nil
			}
			return err
		}
	}
	return nil
}

type WalkFunc = filepath.WalkFunc

var SkipDir = filepath.SkipDir

func Walk(fs FileSystem, root string, walkFn WalkFunc) error {
	info, err := fs.Lstat(root)
	return walkFS(fs, root, info, err, walkFn)
}
