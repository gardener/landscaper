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

package registry

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscaper/pkg/apis/core/install"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

/**

definition.yaml
blob.yaml --> url: xxx

*/

// DefinitionFileName is the filename of a component definition on a local path
const DefinitionFileName = "definition.yaml"

type localRegistry struct {
	log   logr.Logger
	paths []string

	decoder runtime.Decoder

	index Index
}

func NewLocalRegistry(log logr.Logger, paths []string) (Registry, error) {
	lsScheme := runtime.NewScheme()
	if err := install.AddToScheme(lsScheme); err != nil {
		return nil, err
	}

	r := &localRegistry{
		log:     log,
		paths:   paths,
		decoder: serializer.NewCodecFactory(lsScheme).UniversalDecoder(),
		index:   Index{},
	}

	if err := r.setDefinitions(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *localRegistry) setDefinitions() error {
	for _, path := range r.paths {
		index, err := r.findDefinitionsInPath(path)
		if err != nil {
			return err
		}
		r.index.Merge(index)
	}
	return nil
}

// findDefinitionsInPath walks the given path and tries to parse each file with the DefinitionFileName.
// The component definition's directory is set as its corresponding blob.
func (r *localRegistry) findDefinitionsInPath(path string) (Index, error) {
	index := Index{}
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			r.log.Error(err, "unable to walk path", "path", path)
			return nil
		}

		if info.Name() != DefinitionFileName {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			r.log.Error(err, "unable to read file", "path", path)
			return nil
		}

		definition := &v1alpha1.ComponentDefinition{}
		if _, _, err := r.decoder.Decode(data, nil, definition); err != nil {
			r.log.Error(err, "unable to decode ")
			return nil
		}

		index.Add(DefinitionReference{
			SourcePath: path,
			Definition: definition,
			blobPath:   filepath.Dir(path),
		})

		return nil
	})
	return index, err
}

func (r *localRegistry) GetDefinitionByRef(ref string) (*v1alpha1.ComponentDefinition, error) {
	vn, err := ParseDefinitionRef(ref)
	if err != nil {
		return nil, err
	}
	return r.GetDefinition(vn.Name, vn.Version)
}

func (r *localRegistry) GetDefinition(name, version string) (*v1alpha1.ComponentDefinition, error) {
	if _, ok := r.index[name]; !ok {
		return nil, NewComponentNotFoundError(name, nil)
	}
	ref, ok := r.index[name][version]
	if !ok {
		return nil, NewVersionNotFoundError(name, version, nil)
	}
	return ref.Definition, nil
}

// Maybe we should use a virtual filesystem instead of a blob
// and let the registry handle all the blob downloading/compress etc..
// e.g. https://github.com/blang/vfs or https://github.com/spf13/afero which would mean that we would return a vfs.filesystem instead of a byte array
func (r *localRegistry) GetBlob(name, version string) (afero.Fs, error) {
	if _, ok := r.index[name]; !ok {
		return nil, NewComponentNotFoundError(name, nil)
	}
	ref, ok := r.index[name][version]
	if !ok {
		return nil, NewVersionNotFoundError(name, version, nil)
	}

	blobFS := afero.NewBasePathFs(afero.NewOsFs(), ref.blobPath)
	roBlobFS := afero.NewReadOnlyFs(blobFS)

	return roBlobFS, nil
}

func (r *localRegistry) GetVersions(name string) ([]string, error) {
	if _, ok := r.index[name]; !ok {
		return nil, NewComponentNotFoundError(name, nil)
	}
	var (
		versions = make([]string, len(r.index[name]))
		i        = 0
	)
	for version := range r.index[name] {
		versions[i] = version
		i++
	}
	return versions, nil
}

// DefinitionReference is the reference to a local definition
type DefinitionReference struct {
	SourcePath string
	Definition *v1alpha1.ComponentDefinition
	blobPath   string
}

// Index is internal index structure for definition.
// The map indexes name and secondly the version
/**
name:
	version:
		sourcePath: path
		definition: def
		blobPath: def
*/
type Index map[string]map[string]DefinitionReference

// Add adds or updates a definition reference in the index.
func (i Index) Add(ref DefinitionReference) {
	def := ref.Definition
	if _, ok := i[def.Name]; !ok {
		i[def.Name] = make(map[string]DefinitionReference)
	}

	i[def.Name][def.Version] = ref
}

// Merge merges the index a in into the current index.
// Whereas the keys of index a overwrite similar keys of the current index.
func (i Index) Merge(a Index) {
	if len(a) == 0 {
		return
	}

	for name, versionedDefinitions := range a {
		if _, ok := i[name]; !ok {
			i[name] = versionedDefinitions
			continue
		}

		for version, def := range versionedDefinitions {
			i[name][version] = def
		}
	}
}
