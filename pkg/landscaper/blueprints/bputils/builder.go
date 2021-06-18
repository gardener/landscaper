// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bputils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/input"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Builder describes a blueprint builder
type Builder struct {
	bp *lsv1alpha1.Blueprint
	fs vfs.FileSystem
}

// NewBuilder creates a new blueprint builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Blueprint sets the blueprint that should be used for the builder.
func (b *Builder) Blueprint(bp *lsv1alpha1.Blueprint) *Builder {
	b.bp = bp
	return b
}

// Fs sets the filesystem that should be used for the builder.
func (b *Builder) Fs(fs vfs.FileSystem) *Builder {
	b.fs = fs
	return b
}

// BuildBlueprint creates a new blueprint using the given filesystem.
func (b *Builder) BuildBlueprint(fs vfs.FileSystem) error {
	bpBytes, err := json.Marshal(b.bp)
	if err != nil {
		return fmt.Errorf("unable to encode blueprint: %w", err)
	}
	if err := vfs.WriteFile(fs, lsv1alpha1.BlueprintFileName, bpBytes, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write blueprint to filesystem: %w", err)
	}
	return nil
}

// BuildResource uses the configured blueprint and builds a (optionally gzipped) tarred blueprint.
func (b *Builder) BuildResource(compress bool) (io.ReadCloser, error) {
	if b.bp == nil {
		return nil, errors.New("blueprint not set")
	}

	if b.fs == nil {
		b.fs = memoryfs.New()
	}
	if err := b.BuildBlueprint(b.fs); err != nil {
		return nil, err
	}

	blueprintInput := input.BlobInput{
		Type:             input.DirInputType,
		Path:             "",
		CompressWithGzip: &compress,
	}
	blob, err := blueprintInput.Read(b.fs, "/")
	if err != nil {
		return nil, fmt.Errorf("unable to create blob from in memory filesystem: %w", err)
	}
	return blob.Reader, nil
}

// BuildResourceToFs uses the configured blueprint and builds a (optionally gzipped) tarred blueprint.
// The resulting resource is written to the given filesystem and path.
func (b *Builder) BuildResourceToFs(fs vfs.FileSystem, path string, compress bool) error {
	blob, err := b.BuildResource(compress)
	if err != nil {
		return err
	}
	defer blob.Close()

	dir := filepath.Dir(path)
	if _, err := fs.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := fs.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	file, err := fs.Create(path)
	if err != nil {
		return err
	}
	if _, err = io.Copy(file, blob); err != nil {
		return err
	}
	return nil
}
