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
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ComponentDescriptorFileName is the name of the component-descriptor file.
const ComponentDescriptorFileName = "component-descriptor.yaml"

// ArtefactDescriptorFileName is the name of the artefact-descriptor file.
const ArtefactDescriptorFileName = "artefact-descriptor.yaml"

// ManifestFileName is the name of the manifest json file.
const ManifestFileName = "manifest.json"

// BlobsDirectoryName is the name of the blob directory in the tar.
const BlobsDirectoryName = "blobs"

var UnsupportedResolveType = errors.New("UnsupportedResolveType")

var NotFoundError = errors.New("ComponentDescriptorNotFound")

var BlobResolverNotDefinedError = errors.New("BlobResolverNotDefined")

// ComponentResolver describes a general interface to resolve a component descriptor
type ComponentResolver interface {
	Resolve(ctx context.Context, repoCtx v2.Repository, name, version string) (*v2.ComponentDescriptor, error)
	ResolveWithBlobResolver(ctx context.Context, repoCtx v2.Repository, name, version string) (*v2.ComponentDescriptor, BlobResolver, error)
}

// BlobResolver defines a resolver that can fetch
// blobs in a specific context defined in a component descriptor.
type BlobResolver interface {
	Info(ctx context.Context, res v2.Resource) (*BlobInfo, error)
	Resolve(ctx context.Context, res v2.Resource, writer io.Writer) (*BlobInfo, error)
}

// TypedBlobResolver defines a blob resolver
// that is able to resolves a set of access types.
type TypedBlobResolver interface {
	BlobResolver
	// CanResolve returns whether the resolver is able to resolve the
	// resource.
	CanResolve(resource v2.Resource) bool
}

// BlobInfo describes a blob.
type BlobInfo struct {
	// MediaType is the media type of the object this schema refers to.
	MediaType string `json:"mediaType,omitempty"`

	// Digest is the digest of the targeted content.
	Digest string `json:"digest"`

	// Size specifies the size in bytes of the blob.
	Size int64 `json:"size"`
}

// ArchiveFormat describes the format of a component archive.
// A archive can currently be defined in a filesystem, as tar or as gzipped tar.
type ArchiveFormat string

const (
	ArchiveFormatFilesystem ArchiveFormat = "fs"
	ArchiveFormatTar        ArchiveFormat = "tar"
	ArchiveFormatTarGzip    ArchiveFormat = "tgz"
)

type CTF struct {
	fs vfs.FileSystem
	ctfPath string
	tempDir string
	tempFs vfs.FileSystem
}

// NewCTF reads a CTF archive from a file.
// The use should call "Close" to remove all temporary files
func NewCTF(fs vfs.FileSystem, ctfPath string) (*CTF, error) {
	tempDir, err := vfs.TempDir(fs, "", "ctf-")
	if err != nil {
		return nil, err
	}
	tempFs, err := projectionfs.New(fs, tempDir)
	if err != nil {
		return nil, fmt.Errorf("unable to create fs for temporary directory %q: %w", tempDir, err)
	}

	ctf := &CTF{
		fs:      fs,
		ctfPath: ctfPath,
		tempDir: tempDir,
		tempFs:  tempFs,
	}
	if err := ctf.extract(); err != nil {
		return nil, fmt.Errorf("unable to read ctf: %w", err)
	}
	return ctf, nil
}

type WalkFunc = func(ca *ComponentArchive) error

// Walk traverses through all component archives that are included in the ctf.
func (ctf *CTF) Walk(walkFunc WalkFunc) error {
	err := vfs.Walk(ctf.tempFs, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := ctf.tempFs.Open(path)
		if err != nil {
			return fmt.Errorf("unable to read component archive file %q: %w", path, err)
		}
		ca, err := NewComponentArchiveFromTarReader(file)
		if err != nil {
			return err
		}

		return walkFunc(ca)
	})
	return err
}

// AddComponentArchive adds or updates a component archive in the ctf archive.
func (ctf *CTF) AddComponentArchive(ca *ComponentArchive, format ArchiveFormat) error {
	filename, err := ca.Digest()
	if err != nil {
		return err
	}
	return ctf.AddComponentArchiveWithName(filename, ca, format)
}

// AddComponentArchiveWithName adds or updates a component archive in the ctf archive.
// The archive is added to the ctf with the given name
func (ctf *CTF) AddComponentArchiveWithName(filename string, ca *ComponentArchive, format ArchiveFormat) error {
	file, err := ctf.tempFs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	switch format {
	case ArchiveFormatTar:
		if err := ca.WriteTar(file); err != nil {
			_ = file.Close()
			return fmt.Errorf("unable to write component archive to %q: %w", filename, err)
		}
	case ArchiveFormatTarGzip:
		if err := ca.WriteTarGzip(file); err != nil {
			_ = file.Close()
			return fmt.Errorf("unable to write component archive to %q: %w", filename, err)
		}
	default:
		return fmt.Errorf("unsupported archive format %q", format)
	}

	return file.Close()
}

// extract untars the given ctf archive to the tmp directory.
func (ctf *CTF) extract() error {
	file, err := ctf.fs.Open(ctf.ctfPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return ExtractTarToFs(ctf.tempFs, file)
}

// Write writes the current changes back to the original ctf.
func (ctf *CTF) Write() error {
	file, err := ctf.fs.OpenFile(ctf.ctfPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()
	tw := tar.NewWriter(file)
	defer tw.Close()

	err = vfs.Walk(ctf.tempFs, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("unable to write header for %q: %w", path, err)
		}

		blob, err := ctf.tempFs.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open blob %q: %w", path, err)
		}
		defer blob.Close()
		if _, err := io.Copy(tw, blob); err != nil {
			return fmt.Errorf("unable to write blob %q: %w", path, err)
		}
		return nil
	})
	return err
}

// Close closes the CTF that deletes all temporary files
func (ctf *CTF) Close() error {
	return ctf.fs.RemoveAll(ctf.tempDir)
}

// AggregatedBlobResolver combines multiple blob resolver.
// Is automatically picks the right resolver based on the resolvers type information.
// If multiple resolvers match, the first matching resolver is used.
type AggregatedBlobResolver struct {
	resolver []TypedBlobResolver
}

var _ BlobResolver = &AggregatedBlobResolver{}

// NewAggregatedBlobResolver creates a new aggregated resolver.
// Note that only typed resolvers can be added.
// An error is thrown if a resolver does not implement the supported types.
func NewAggregatedBlobResolver(resolvers ...BlobResolver) (*AggregatedBlobResolver, error) {
	agg := &AggregatedBlobResolver{
		resolver: make([]TypedBlobResolver, 0),
	}
	if err := agg.Add(resolvers...); err != nil {
		return nil, err
	}
	return agg, nil
}

// Add adds multiple resolvers to the aggregator.
// Only typed resolvers can be added.
// An error is thrown if a resolver does not implement the supported types function.
func (a *AggregatedBlobResolver) Add(resolvers ...BlobResolver) error {
	for i, resolver := range resolvers {
		typedResolver, ok := resolver.(TypedBlobResolver)
		if !ok {
			return fmt.Errorf("resolver %d does not implement supported types interface", i)
		}
		a.resolver = append(a.resolver, typedResolver)
	}
	return nil
}

func (a *AggregatedBlobResolver) Info(ctx context.Context, res v2.Resource) (*BlobInfo, error) {
	resolver, err := a.getResolver(res)
	if err != nil {
		return nil, err
	}
	return resolver.Info(ctx, res)
}

func (a *AggregatedBlobResolver) Resolve(ctx context.Context, res v2.Resource, writer io.Writer) (*BlobInfo, error) {
	resolver, err := a.getResolver(res)
	if err != nil {
		return nil, err
	}
	return resolver.Resolve(ctx, res, writer)
}

func (a *AggregatedBlobResolver) getResolver(res v2.Resource) (BlobResolver, error) {
	if res.Access == nil {
		return nil, fmt.Errorf("no access is defined")
	}

	for _, resolver := range a.resolver {
		if resolver.CanResolve(res) {
			return resolver, nil
		}
	}
	return nil, UnsupportedResolveType
}

// AggregateBlobResolvers aggregates two resolvers to one by using aggregated blob resolver.
func AggregateBlobResolvers(a, b BlobResolver) (BlobResolver, error) {
	aggregated, ok := a.(*AggregatedBlobResolver)
	if ok {
		if err := aggregated.Add(b); err != nil {
			return nil, fmt.Errorf("unable to add second resolver to aggreagted first resolver: %w", err)
		}
		return aggregated, nil
	}

	aggregated, ok = b.(*AggregatedBlobResolver)
	if ok {
		if err := aggregated.Add(a); err != nil {
			return nil, fmt.Errorf("unable to add first resolver to aggreagted second resolver: %w", err)
		}
		return aggregated, nil
	}

	// create a new aggregated resolver if neither a nor b are aggregations
	return NewAggregatedBlobResolver(a, b)
}
