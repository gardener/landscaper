// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessobj

import (
	"archive/tar"
	"io"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/errors"
)

func mapErr(forced bool, err error) (bool, bool, error) {
	if !forced {
		return false, false, nil
	}
	return false, true, err
}

// CheckFile returns create, acceptable, error.
func CheckFile(kind string, createHint string, forcedType bool, path string, fs vfs.FileSystem, descriptorname string) (bool, bool, error) {
	info, err := fs.Stat(path)
	if err != nil {
		if createHint == kind {
			if vfs.IsErrNotExist(err) {
				return true, true, nil
			}
		}
		return mapErr(forcedType, err)
	}
	accepted := false
	if !info.IsDir() {
		file, err := fs.Open(path)
		if err != nil {
			return mapErr(forcedType, err)
		}
		defer file.Close()
		forcedType = false
		r, _, err := compression.AutoDecompress(file)
		if err != nil {
			return mapErr(forcedType, err)
		}
		tr := tar.NewReader(r)
		for {
			header, err := tr.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return mapErr(forcedType, err)
			}

			switch header.Typeflag {
			case tar.TypeReg:
				if header.Name == descriptorname {
					accepted = true
					break
				}
			}
		}
	} else {
		if forcedType {
			entries, err := vfs.ReadDir(fs, path)
			if err == nil && len(entries) > 0 {
				forcedType = false
			}
		}
		if ok, err := vfs.FileExists(fs, filepath.Join(path, descriptorname)); !ok || err != nil {
			if err != nil {
				return mapErr(forcedType, err)
			}
		} else {
			accepted = ok
		}
	}
	if !accepted {
		return mapErr(forcedType, errors.Newf("%s: no %s", path, kind))
	}
	return false, true, nil
}
