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

package ctf

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
)

// NewComponentArchive returns a new component descriptor with a filesystem
func NewComponentArchive(cd *v2.ComponentDescriptor, fs vfs.FileSystem) *ComponentArchive {
	return &ComponentArchive{
		ComponentDescriptor: cd,
		fs:                  fs,
		BlobResolver: &ComponentArchiveBlobResolver{
			fs: fs,
		},
	}
}

// ComponentArchiveFromPath creates a component archive from a path
func ComponentArchiveFromPath(path string) (*ComponentArchive, error) {
	fs, err := projectionfs.New(osfs.New(), path)
	if err != nil {
		return nil, fmt.Errorf("unable to create projected filesystem from path %s: %w", path, err)
	}

	return NewComponentArchiveFromFilesystem(fs)
}

// ComponentArchiveFromCompressedCTF creates a new component archive from a zipped CTF tar.
func ComponentArchiveFromCompressedCTF(path string) (*ComponentArchive, error) {
	// we expect that the path point to a targz
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open tar archive from %s: %w", path, err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("unable to open gzip reader for %s: %w", path, err)
	}
	return NewComponentArchiveFromTarReader(reader)
}

// ComponentArchiveFromCTF creates a new componet archive from a CTF tar file.
func ComponentArchiveFromCTF(path string) (*ComponentArchive, error) {
	// we expect that the path point to a tar
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open tar archive from %s: %w", path, err)
	}
	defer file.Close()
	return NewComponentArchiveFromTarReader(file)
}

// NewComponentArchiveFromTarReader creates a new manifest builder from a input reader.
// todo: make the fs configurable to also use a temporary filesystem
func NewComponentArchiveFromTarReader(in io.Reader) (*ComponentArchive, error) {
	// the archive is untared to a memory fs that the builder can work
	// as it would be a default filesystem.
	fs := memoryfs.New()
	if err := ExtractTarToFs(fs, in); err != nil {
		return nil, fmt.Errorf("unable to extract tar: %w", err)
	}

	return NewComponentArchiveFromFilesystem(fs)
}

// NewComponentArchiveFromFilesystem creates a new component archive from a filesystem.
func NewComponentArchiveFromFilesystem(fs vfs.FileSystem, decodeOpts ...codec.DecodeOption) (*ComponentArchive, error) {
	data, err := vfs.ReadFile(fs, filepath.Join("/", ComponentDescriptorFileName))
	if err != nil {
		return nil, fmt.Errorf("unable to read the component descriptor from %s: %w", ComponentDescriptorFileName, err)
	}
	cd := &v2.ComponentDescriptor{}
	if err := codec.Decode(data, cd, decodeOpts...); err != nil {
		return nil, fmt.Errorf("unable to parse component descriptor read from %s: %w", ComponentDescriptorFileName, err)
	}

	return &ComponentArchive{
		ComponentDescriptor: cd,
		fs:                  fs,
		BlobResolver: &ComponentArchiveBlobResolver{
			fs: fs,
		},
	}, nil
}

// ComponentArchive is the go representation for a CTF component artefact
type ComponentArchive struct {
	ComponentDescriptor *v2.ComponentDescriptor
	fs                  vfs.FileSystem
	BlobResolver
}

// Digest returns the digest of the component archive.
// The digest is computed serializing the included component descriptor into json and compute sha hash.
func (ca *ComponentArchive) Digest() (string, error) {
	data, err := codec.Encode(ca.ComponentDescriptor)
	if err != nil {
		return "", err
	}
	return digest.FromBytes(data).String(), nil
}

// AddResource adds a blob resource to the current archive.
// If the specified resource already exists it will be overwritten.
func (ca *ComponentArchive) AddResource(res *v2.Resource, info BlobInfo, reader io.Reader) error {
	if res == nil {
		return errors.New("a resource has to be defined")
	}
	id := ca.ComponentDescriptor.GetResourceIndex(*res)
	if err := ca.ensureBlobsPath(); err != nil {
		return err
	}

	blobpath := BlobPath(info.Digest)
	if _, err := ca.fs.Stat(blobpath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to get file info for %s", blobpath)
		}
		file, err := ca.fs.OpenFile(blobpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open file %s: %w", blobpath, err)
		}
		if _, err := io.Copy(file, reader); err != nil {
			return fmt.Errorf("unable to write blob to file: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("unable to close file: %w", err)
		}
	}

	localFsAccess := v2.NewLocalFilesystemBlobAccess(info.Digest, info.MediaType)
	unstructuredType, err := v2.NewUnstructured(localFsAccess)
	if err != nil {
		return fmt.Errorf("unable to convert local filesystem type to untructured type: %w", err)
	}
	res.Access = &unstructuredType

	if id == -1 {
		ca.ComponentDescriptor.Resources = append(ca.ComponentDescriptor.Resources, *res)
	} else {
		ca.ComponentDescriptor.Resources[id] = *res
	}
	return nil
}

// AddSource adds a blob source to the current archive.
// If the specified source already exists it will be overwritten.
func (ca *ComponentArchive) AddSource(src *v2.Source, info BlobInfo, reader io.Reader) error {
	if src == nil {
		return errors.New("a source has to be defined")
	}
	id := ca.ComponentDescriptor.GetSourceIndex(*src)
	if err := ca.ensureBlobsPath(); err != nil {
		return err
	}

	blobpath := BlobPath(info.Digest)
	if _, err := ca.fs.Stat(blobpath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to get file info for %s", blobpath)
		}
		file, err := ca.fs.OpenFile(blobpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open file %s: %w", blobpath, err)
		}
		if _, err := io.Copy(file, reader); err != nil {
			return fmt.Errorf("unable to write blob to file: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("unable to close file: %w", err)
		}
	}

	localFsAccess := v2.NewLocalFilesystemBlobAccess(info.Digest, info.MediaType)
	unstructuredType, err := v2.NewUnstructured(localFsAccess)
	if err != nil {
		return fmt.Errorf("unable to convert local filesystem type to untructured type: %w", err)
	}
	src.Access = &unstructuredType

	if id == -1 {
		ca.ComponentDescriptor.Sources = append(ca.ComponentDescriptor.Sources, *src)
	} else {
		ca.ComponentDescriptor.Sources[id] = *src
	}
	return nil
}

// AddResourceFromResolver adds a blob resource to the current archive.
// If the specified resource already exists it will be overwritten.
func (ca *ComponentArchive) AddResourceFromResolver(ctx context.Context, res *v2.Resource, resolver BlobResolver) error {
	if res == nil {
		return errors.New("a resource has to be defined")
	}
	id := ca.ComponentDescriptor.GetResourceIndex(*res)
	if err := ca.ensureBlobsPath(); err != nil {
		return err
	}

	info, err := resolver.Info(ctx, *res)
	if err != nil {
		return fmt.Errorf("unable to get blob info from resolver: %w", err)
	}

	blobpath := BlobPath(info.Digest)
	if _, err := ca.fs.Stat(blobpath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to get file info for %s", blobpath)
		}
		file, err := ca.fs.OpenFile(blobpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open file %s: %w", blobpath, err)
		}
		if _, err := resolver.Resolve(ctx, *res, file); err != nil {
			return fmt.Errorf("unable to write blob to file: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("unable to close file: %w", err)
		}
	}

	localFsAccess := v2.NewLocalFilesystemBlobAccess(info.Digest, info.MediaType)
	unstructuredType, err := v2.NewUnstructured(localFsAccess)
	if err != nil {
		return fmt.Errorf("unable to convert local filesystem type to untructured type: %w", err)
	}
	res.Access = &unstructuredType

	if id == -1 {
		ca.ComponentDescriptor.Resources = append(ca.ComponentDescriptor.Resources, *res)
	} else {
		ca.ComponentDescriptor.Resources[id] = *res
	}
	return nil
}

// ensureBlobsPath ensures that the blob directory exists
func (ca *ComponentArchive) ensureBlobsPath() error {
	if _, err := ca.fs.Stat(BlobsDirectoryName); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to get file info for blob directory: %w", err)
		}
		return ca.fs.Mkdir(BlobsDirectoryName, os.ModePerm)
	}
	return nil
}

// WriteTarGzip tars the current components descriptor and its artifacts.
func (ca *ComponentArchive) WriteTarGzip(writer io.Writer) error {
	gw := gzip.NewWriter(writer)
	if err := ca.WriteTar(gw); err != nil {
		return err
	}
	return gw.Close()
}

// WriteTar tars the current components descriptor and its artifacts.
func (ca *ComponentArchive) WriteTar(writer io.Writer) error {
	tw := tar.NewWriter(writer)

	// write component descriptor
	cdBytes, err := codec.Encode(ca.ComponentDescriptor)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor: %w", err)
	}
	cdHeader := &tar.Header{
		Name:    ComponentDescriptorFileName,
		Size:    int64(len(cdBytes)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(cdHeader); err != nil {
		return fmt.Errorf("unable to write component descriptor header: %w", err)
	}
	if _, err := io.Copy(tw, bytes.NewBuffer(cdBytes)); err != nil {
		return fmt.Errorf("unable to write component descriptor content: %w", err)
	}

	// add all blobs
	err = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     BlobsDirectoryName,
		Mode:     0644,
		ModTime:  time.Now(),
	})
	if err != nil {
		return fmt.Errorf("unable to write blob directory: %w", err)
	}

	blobs, err := vfs.ReadDir(ca.fs, BlobsDirectoryName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to read blob directory: %w", err)
	}
	for _, blobInfo := range blobs {
		blobpath := BlobPath(blobInfo.Name())
		header := &tar.Header{
			Name:    blobpath,
			Size:    blobInfo.Size(),
			Mode:    0644,
			ModTime: time.Now(),
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("unable to write blob header: %w", err)
		}

		blob, err := ca.fs.Open(blobpath)
		if err != nil {
			return fmt.Errorf("unable to open blob: %w", err)
		}
		if _, err := io.Copy(tw, blob); err != nil {
			return fmt.Errorf("unable to write blob content: %w", err)
		}
		if err := blob.Close(); err != nil {
			return fmt.Errorf("unable to close blob %s: %w", blobpath, err)
		}
	}

	return tw.Close()
}

// WriteToFilesystem writes the current component archive to a filesystem
func (ca *ComponentArchive) WriteToFilesystem(fs vfs.FileSystem, path string) error {
	// create the directory structure with the blob directory
	if err := fs.MkdirAll(filepath.Join(path, BlobsDirectoryName), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create output directory %q: %s", path, err.Error())
	}
	// copy component-descriptor
	cdBytes, err := codec.Encode(ca.ComponentDescriptor)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor: %w", err)
	}
	if err := vfs.WriteFile(fs, filepath.Join(path, ComponentDescriptorFileName), cdBytes, os.ModePerm); err != nil {
		return fmt.Errorf("unable to copy component descritptor to %q: %w", filepath.Join(path, ComponentDescriptorFileName), err)
	}

	// copy all blobs
	blobInfos, err := vfs.ReadDir(ca.fs, BlobsDirectoryName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to read blobs: %w", err)
	}
	for _, blobInfo := range blobInfos {
		if blobInfo.IsDir() {
			continue
		}
		inpath := BlobPath(blobInfo.Name())
		outpath := filepath.Join(path, BlobsDirectoryName, blobInfo.Name())
		blob, err := ca.fs.Open(inpath)
		if err != nil {
			return fmt.Errorf("unable to open input blob %q: %w", inpath, err)
		}
		out, err := fs.OpenFile(outpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to open output blob %q: %w", outpath, err)
		}
		if _, err := io.Copy(out, blob); err != nil {
			return fmt.Errorf("unable to copy blob from %q to %q: %w", inpath, outpath, err)
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("unable to close output blob %s: %w", outpath, err)
		}
		if err := blob.Close(); err != nil {
			return fmt.Errorf("unable to close input blob %s: %w", outpath, err)
		}
	}

	return nil
}

// ComponentArchiveBlobResolver implements the BlobResolver interface for
// "LocalFilesystemBlob" access types.
type ComponentArchiveBlobResolver struct {
	fs vfs.FileSystem
}

// NewComponentArchiveBlobResolver creates new ComponentArchive blob that can resolve local filesystem references.
// The filesystem is expected to have its root at the component archives root
// so that artifacts can be resolve in "/blobs".
func NewComponentArchiveBlobResolver(fs vfs.FileSystem) *ComponentArchiveBlobResolver {
	return &ComponentArchiveBlobResolver{
		fs: fs,
	}
}

func (ca *ComponentArchiveBlobResolver) CanResolve(res v2.Resource) bool {
	return res.Access != nil && res.Access.GetType() == v2.LocalFilesystemBlobType
}

func (ca *ComponentArchiveBlobResolver) Info(ctx context.Context, res v2.Resource) (*BlobInfo, error) {
	info, file, err := ca.resolve(ctx, res)
	if err != nil {
		return nil, err
	}
	if file != nil {
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	return info, nil
}

// Resolve fetches the blob for a given resource and writes it to the given tar.
func (ca *ComponentArchiveBlobResolver) Resolve(ctx context.Context, res v2.Resource, writer io.Writer) (*BlobInfo, error) {
	info, file, err := ca.resolve(ctx, res)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(writer, file); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return info, nil
}

func (ca *ComponentArchiveBlobResolver) resolve(_ context.Context, res v2.Resource) (*BlobInfo, vfs.File, error) {
	if res.Access == nil || res.Access.GetType() != v2.LocalFilesystemBlobType {
		return nil, nil, UnsupportedResolveType
	}
	localFSAccess := &v2.LocalFilesystemBlobAccess{}
	if err := res.Access.DecodeInto(localFSAccess); err != nil {
		return nil, nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
	}
	blobpath := BlobPath(localFSAccess.Filename)
	mediaType := res.GetType()
	if len(localFSAccess.MediaType) != 0 {
		mediaType = localFSAccess.MediaType
	}

	info, err := ca.fs.Stat(blobpath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get fileinfo for %s: %w", blobpath, err)
	}
	if info.IsDir() {
		return nil, nil, fmt.Errorf("directories are not allowed as blobs %s", blobpath)
	}
	file, err := ca.fs.Open(blobpath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open blob from %s", blobpath)
	}

	dig, err := digest.FromReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate dig from %s: %w", localFSAccess.Filename, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, fmt.Errorf("unable to reset file reader: %w", err)
	}
	return &BlobInfo{
		MediaType: mediaType,
		Digest:    dig.String(),
		Size:      info.Size(),
	}, file, nil
}

// BlobPath returns the path to the blob for a given name.
func BlobPath(name string) string {
	return filepath.Join(BlobsDirectoryName, name)
}

// ExtractTarToFs writes a tar stream to a filesystem.
func ExtractTarToFs(fs vfs.FileSystem, in io.Reader) error {
	tr := tar.NewReader(in)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(header.Name, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("unable to create directory %s: %w", header.Name, err)
			}
		case tar.TypeReg:
			file, err := fs.OpenFile(header.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("unable to open file %s: %w", header.Name, err)
			}
			if _, err := io.Copy(file, tr); err != nil {
				return fmt.Errorf("unable to copy tar file to filesystem: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("unable to close file %s: %w", header.Name, err)
			}
		}
	}
}
