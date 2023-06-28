// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tarutils

import (
	"archive/tar"
	"fmt"
	"io"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

// ExtractArchiveToFs wunpacks an archive to a filesystem.
func ExtractArchiveToFs(fs vfs.FileSystem, path string, fss ...vfs.FileSystem) error {
	sfs := utils.OptionalDefaulted(osfs.New(), fss...)

	f, err := sfs.Open(path)
	if err != nil {
		return errors.Wrapf(err, "cannot open %s", path)
	}
	defer f.Close()
	r, _, err := compression.AutoDecompress(f)
	if err != nil {
		return errors.Wrapf(err, "cannot determine compression for %s", path)
	}
	return ExtractTarToFs(fs, r)
}

// ExtractArchiveToFsWithInfo wunpacks an archive to a filesystem.
func ExtractArchiveToFsWithInfo(fs vfs.FileSystem, path string, fss ...vfs.FileSystem) (int64, int64, error) {
	sfs := utils.OptionalDefaulted(osfs.New(), fss...)

	f, err := sfs.Open(path)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "cannot open %s", path)
	}
	defer f.Close()
	r, _, err := compression.AutoDecompress(f)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "cannot determine compression for %s", path)
	}
	return ExtractTarToFsWithInfo(fs, r)
}

// ExtractTarToFs writes a tar stream to a filesystem.
func ExtractTarToFs(fs vfs.FileSystem, in io.Reader) error {
	_, _, err := ExtractTarToFsWithInfo(fs, in)
	return err
}

func ExtractTarToFsWithInfo(fs vfs.FileSystem, in io.Reader) (fcnt int64, bcnt int64, err error) {
	tr := tar.NewReader(in)
	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return fcnt, bcnt, nil
			}
			return fcnt, bcnt, err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(header.Name, vfs.FileMode(header.Mode)); err != nil {
				return fcnt, bcnt, fmt.Errorf("unable to create directory %s: %w", header.Name, err)
			}
		case tar.TypeSymlink, tar.TypeLink:
			dir := vfs.Dir(fs, header.Name)
			if err := fs.MkdirAll(dir, 0o766); err != nil {
				return fcnt, bcnt, fmt.Errorf("unable to create directory %s: %w", dir, err)
			}
			err := fs.Symlink(header.Linkname, header.Name)
			if err != nil {
				return fcnt, bcnt, fmt.Errorf("unable to create symbolic link %s: %w", header.Name, err)
			}
			fcnt++
		case tar.TypeReg:
			dir := vfs.Dir(fs, header.Name)
			if err := fs.MkdirAll(dir, 0o766); err != nil {
				return fcnt, bcnt, fmt.Errorf("unable to create directory %s: %w", dir, err)
			}
			file, err := fs.OpenFile(header.Name, vfs.O_WRONLY|vfs.O_CREATE|vfs.O_TRUNC, vfs.FileMode(header.Mode))
			if err != nil {
				return fcnt, bcnt, fmt.Errorf("unable to open file %s: %w", header.Name, err)
			}
			bcnt += header.Size
			//nolint:gosec // We don't know what size limit we could set, the tar
			// archive can be an image layer and that can even reach the gigabyte range.
			// For now, we acknowledge the risk.
			//
			// We checked other softwares and tried to figure out how they manage this,
			// but it's handled the same way.
			if _, err := io.Copy(file, tr); err != nil {
				return fcnt, bcnt, fmt.Errorf("unable to copy tar file to filesystem: %w", err)
			}
			if err := file.Close(); err != nil {
				return fcnt, bcnt, fmt.Errorf("unable to close file %s: %w", header.Name, err)
			}
			fcnt++
		}
	}
}
