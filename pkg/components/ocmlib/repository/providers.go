// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/utils"
)

type FilesystemCompDescProvider struct {
	CompDescFs      vfs.FileSystem
	CompDescDirPath string
}

func NewFilesystemCompDescProvider(path string, fs ...vfs.FileSystem) ComponentDescriptorProvider {
	return &FilesystemCompDescProvider{
		CompDescDirPath: path,
		CompDescFs:      utils.Optional(fs...),
	}
}

func (f *FilesystemCompDescProvider) List() ([]*compdesc.ComponentDescriptor, error) {
	p := "/"
	if f.CompDescDirPath != "" {
		p = f.CompDescDirPath
	}
	fs := f.CompDescFs
	if fs == nil {
		fs = osfs.New()
	}
	var result []*compdesc.ComponentDescriptor
	entries, err := vfs.ReadDir(fs, p)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		filename := filepath.Join(p, e.Name())
		// there might e.g. be a blob directory in this folder
		if ok, err := vfs.IsDir(fs, filename); ok || err != nil {
			continue
		}
		data, err := vfs.ReadFile(fs, filename)
		if err != nil {
			return nil, err
		}
		cd, err := compdesc.Decode(data)
		if err != nil {
			return nil, err
		}
		result = append(result, cd)
	}
	return result, nil
}

type MemoryCompDescProvider struct {
	CompDescs []*compdesc.ComponentDescriptor
}

func NewMemoryCompDescProvider(descriptors []*compdesc.ComponentDescriptor) ComponentDescriptorProvider {
	return &MemoryCompDescProvider{
		CompDescs: descriptors,
	}
}

func (m *MemoryCompDescProvider) List() ([]*compdesc.ComponentDescriptor, error) {
	return m.CompDescs, nil
}
