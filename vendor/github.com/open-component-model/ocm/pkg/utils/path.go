// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

func ResolvePath(path string) (string, error) {
	if strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		home := os.Getenv("HOME")
		if home == "" {
			return path, fmt.Errorf("HOME not set")
		}
		path = home + path[1:]
	}
	return path, nil
}

func ReadFile(fs vfs.FileSystem, path string) ([]byte, error) {
	path, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}
	return vfs.ReadFile(fs, path)
}
