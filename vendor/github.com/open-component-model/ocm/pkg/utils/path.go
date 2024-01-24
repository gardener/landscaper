// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/errors"
)

// ResolvePath handles the ~ notation for the home directory.
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

func ResolveData(in string, fss ...vfs.FileSystem) ([]byte, error) {
	return handlePrefix(func(in string, fs vfs.FileSystem) ([]byte, error) { return []byte(in), nil }, in, fss...)
}

func ReadFile(in string, fss ...vfs.FileSystem) ([]byte, error) {
	return handlePrefix(readFile, in, fss...)
}

func handlePrefix(def func(string, vfs.FileSystem) ([]byte, error), in string, fss ...vfs.FileSystem) ([]byte, error) {
	if strings.HasPrefix(in, "=") {
		return []byte(in[1:]), nil
	}
	if strings.HasPrefix(in, "!") {
		return base64.StdEncoding.DecodeString(in[1:])
	}
	if strings.HasPrefix(in, "@") {
		return readFile(in[1:], FileSystem(fss...))
	}
	return def(in, FileSystem(fss...))
}

func readFile(path string, fs vfs.FileSystem) ([]byte, error) {
	path, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}
	data, err := vfs.ReadFile(fs, path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read file %q", path)
	}
	return data, nil
}

var _osfs = osfs.New()

func FileSystem(fss ...vfs.FileSystem) vfs.FileSystem {
	return DefaultedFileSystem(_osfs, fss...)
}

func DefaultedFileSystem(def vfs.FileSystem, fss ...vfs.FileSystem) vfs.FileSystem {
	return OptionalDefaulted(def, fss...)
}
