// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// BuildTarGzip creates a new compressed tar based on a filesystem and a path.
// The tar is written to the given io.Writer.
func BuildTarGzip(fs vfs.FileSystem, root string, buf io.Writer) error {
	zr := gzip.NewWriter(buf)
	if err := BuildTar(fs, root, zr); err != nil {
		return err
	}
	return zr.Close()
}

// BuildTar creates a new tar based on a filesystem and a path.
// The tar is written to the given io.Writer.
func BuildTar(fs vfs.FileSystem, root string, buf io.Writer) error {
	tw := tar.NewWriter(buf)
	err := vfs.Walk(fs, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && len(info.Name()) == 0 {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return err
		}

		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		data, err := fs.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, data); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}

	return nil
}

// ExtractTarGzip extracts the content of a tar to the given filesystem with the given root base path
func ExtractTarGzip(gzipStream io.Reader, fs vfs.FileSystem, root string) error {
	uncompStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(uncompStream)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(path.Join(root, header.Name), os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			file, err := fs.Create(path.Join(root, header.Name))
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				if err := file.Close(); err != nil {
					return err
				}
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}
		}
	}
}

// BuildTarGzipLayer tar and gzips the given path and adds the layer to the cache.
// It returns the newly creates ocispec Description for the tar.
func BuildTarGzipLayer(cache cache.Cache, fs vfs.FileSystem, path string, annotations map[string]string) (ocispecv1.Descriptor, error) {

	var blob bytes.Buffer
	if err := BuildTarGzip(fs, path, &blob); err != nil {
		return ocispecv1.Descriptor{}, err
	}

	desc := ocispecv1.Descriptor{
		MediaType:   ociclient.MediaTypeTarGzip,
		Digest:      digest.FromBytes(blob.Bytes()),
		Size:        int64(blob.Len()),
		Annotations: annotations,
	}

	if err := cache.Add(desc, ioutil.NopCloser(&blob)); err != nil {
		return ocispecv1.Descriptor{}, err
	}

	return desc, nil
}
