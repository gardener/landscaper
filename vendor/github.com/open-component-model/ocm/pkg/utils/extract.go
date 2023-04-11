// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

// ExtractTarToFs writes a tar stream to a filesystem.
func ExtractTarToFs(fs vfs.FileSystem, in io.Reader) error {
	tr := tar.NewReader(in)
	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(header.Name, vfs.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("unable to create directory %s: %w", header.Name, err)
			}
		case tar.TypeReg:
			dir := filepath.Dir(header.Name)
			if err := fs.MkdirAll(dir, 0o766); err != nil {
				return fmt.Errorf("unable to create directory %s: %w", dir, err)
			}
			file, err := fs.OpenFile(header.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, vfs.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("unable to open file %s: %w", header.Name, err)
			}
			//nolint:gosec // We don't know what size limit we could set, the tar
			// archive can be an image layer and that can even reach the gigabyte range.
			// For now, we acknowledge the risk.
			//
			// We checked other softwares and tried to figure out how they manage this,
			// but it's handled the same way.
			if _, err := io.Copy(file, tr); err != nil {
				return fmt.Errorf("unable to copy tar file to filesystem: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("unable to close file %s: %w", header.Name, err)
			}
		}
	}
}
