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

package executions

import (
	"os"

	"github.com/spf13/afero"
)

// LandscaperTplFuncMap contains all additional landscaper functions that are
// available in the executors templates.
func LandscaperTplFuncMap(fs afero.Fs) map[string]interface{} {
	return map[string]interface{}{
		"readFile": readFileFunc(fs),
		"readDir": readDir(fs),
	}
}

// readFileFunc returns a function that reads a file from a location in a filesystem
func readFileFunc(fs afero.Fs) func(path string) []byte{
	return func(path string) []byte {
		file, err := afero.ReadFile(fs, path)
		if err != nil {
			// maybe we should ignore the error and return an empty byte array
			panic(err)
		}
		return file
	}
}

// readDir lists all files of directory
func readDir(fs afero.Fs) func(path string) []os.FileInfo {
	return func(path string) []os.FileInfo {
		files, err := afero.ReadDir(fs, path)
		if err != nil {
			// maybe we should ignore the error and return an empty byte array
			panic(err)
		}
		return files
	}
}
