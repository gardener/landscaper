// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package input

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mandelsoft/vfs/pkg/composefs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"

	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
)

// MediaTypeTar defines the media type for a tarred file
const MediaTypeTar = "application/x-tar"

// MediaTypeGZip defines the media type for a gzipped file
const MediaTypeGZip = "application/gzip"

// MediaTypeOctetStream is the media type for any binary data.
const MediaTypeOctetStream = "application/octet-stream"

// BlobOutput is the output if read BlobInput.
type BlobOutput struct {
	Digest string
	Size   int64
	Reader io.ReadCloser
}

type BlobInputType string

const (
	FileInputType = "file"
	DirInputType  = "dir"
)

// BlobInput defines a local resource input that should be added to the component descriptor and
// to the resource's access.
type BlobInput struct {
	// Type defines the input type of the blob to be added.
	// Note that a input blob of type "dir" is automatically tarred.
	Type BlobInputType `json:"type"`
	// MediaType is the mediatype of the defined file that is also added to the oci layer.
	// Should be a custom media type in the form of "application/vnd.<mydomain>.<my description>"
	MediaType string `json:"mediaType,omitempty"`
	// Path is the path that points to the blob to be added.
	Path string `json:"path"`
	// CompressWithGzip defines that the blob should be automatically compressed using gzip.
	CompressWithGzip *bool `json:"compress,omitempty"`
	// PreserveDir defines that the directory specified in the Path field should be included in the blob.
	// Only supported for Type dir.
	PreserveDir bool `json:"preserveDir,omitempty"`
}

// Compress returns if the blob should be compressed using gzip.
func (input BlobInput) Compress() bool {
	if input.CompressWithGzip == nil {
		return false
	}
	return *input.CompressWithGzip
}

// SetMediaTypeIfNotDefined sets the media type of the input blob if its not defined
func (input *BlobInput) SetMediaTypeIfNotDefined(mediaType string) {
	if len(input.MediaType) != 0 {
		return
	}
	input.MediaType = mediaType
}

// Read reads the configured blob and returns a reader to the given file.
func (input *BlobInput) Read(fs vfs.FileSystem, inputFilePath string) (*BlobOutput, error) {
	inputPath := input.Path
	if !filepath.IsAbs(input.Path) {
		var wd string
		if len(inputFilePath) == 0 {
			// default to working directory if now input filepath is given
			var err error
			wd, err = os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("unable to read current working directory: %w", err)
			}
		} else {
			wd = filepath.Dir(inputFilePath)
		}
		inputPath = filepath.Join(wd, input.Path)
	}
	inputInfo, err := fs.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get info for input blob from %q, %w", inputPath, err)
	}

	// automatically tar the input artifact if it is a directory
	if input.Type == DirInputType {
		if !inputInfo.IsDir() {
			return nil, fmt.Errorf("resource type is dir but a file was provided")
		}
		blobFs, err := projectionfs.New(fs, inputPath)
		if err != nil {
			return nil, fmt.Errorf("unable to create internal fs for %q: %w", inputPath, err)
		}

		if input.PreserveDir {
			dir := string(filepath.Separator) + filepath.Base(inputPath)
			fs := memoryfs.New()
			err = fs.Mkdir(dir, os.FileMode(0777))
			if err != nil {
				return nil, err
			}

			composedFs := composefs.New(fs)
			err = composedFs.Mount(dir, blobFs)
			if err != nil {
				return nil, err
			}

			blobFs = composedFs
		}

		var (
			data bytes.Buffer
		)
		if input.Compress() {
			input.SetMediaTypeIfNotDefined(MediaTypeGZip)
			gw := gzip.NewWriter(&data)
			if err := TarFileSystem(blobFs, gw); err != nil {
				return nil, fmt.Errorf("unable to tar input artifact: %w", err)
			}
			if err := gw.Close(); err != nil {
				return nil, fmt.Errorf("unable to close gzip writer: %w", err)
			}
		} else {
			input.SetMediaTypeIfNotDefined(MediaTypeTar)
			if err := TarFileSystem(blobFs, &data); err != nil {
				return nil, fmt.Errorf("unable to tar input artifact: %w", err)
			}
		}

		return &BlobOutput{
			Digest: digest.FromBytes(data.Bytes()).String(),
			Size:   int64(data.Len()),
			Reader: ioutil.NopCloser(&data),
		}, nil
	} else if input.Type == FileInputType {
		if inputInfo.IsDir() {
			return nil, fmt.Errorf("resource type is file but a directory was provided")
		}
		// otherwise just open the file
		inputBlob, err := fs.Open(inputPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read input blob from %q: %w", inputPath, err)
		}
		blobDigest, err := digest.FromReader(inputBlob)
		if err != nil {
			return nil, fmt.Errorf("unable to calculate digest for input blob from %q, %w", inputPath, err)
		}
		if _, err := inputBlob.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("unable to reset input file: %s", err)
		}

		if input.Compress() {
			input.SetMediaTypeIfNotDefined(MediaTypeGZip)
			var data bytes.Buffer
			gw := gzip.NewWriter(&data)
			if _, err := io.Copy(gw, inputBlob); err != nil {
				return nil, fmt.Errorf("unable to compress input file %q: %w", inputPath, err)
			}
			if err := gw.Close(); err != nil {
				return nil, fmt.Errorf("unable to close gzip writer: %w", err)
			}

			return &BlobOutput{
				Digest: digest.FromBytes(data.Bytes()).String(),
				Size:   int64(data.Len()),
				Reader: ioutil.NopCloser(&data),
			}, nil
		}
		return &BlobOutput{
			Digest: blobDigest.String(),
			Size:   inputInfo.Size(),
			Reader: inputBlob,
		}, nil
	} else {
		return nil, fmt.Errorf("unknown input type %q", inputPath)
	}
}

// TarFileSystem creates a tar archive from a filesystem.
func TarFileSystem(fs vfs.FileSystem, writer io.Writer) error {
	tw := tar.NewWriter(writer)

	err := vfs.Walk(fs, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel("/", path)
		if err != nil {
			return fmt.Errorf("unable to calculate relative path for %s: %w", path, err)
		}
		// ignore the root directory.
		if relPath == "." {
			return nil
		}
		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("unable to write header for %q: %w", path, err)
		}
		if info.IsDir() {
			return nil
		}

		file, err := fs.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open file %q: %w", path, err)
		}
		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("unable to add file to tar %q: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("unable to close file %q: %w", path, err)
		}
		return nil
	})
	return err
}
