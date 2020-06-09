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

package fake

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
)

// FakeRegistry is a fake registry.Registry implementation to mock a registry for testing
type FakeRegistry struct {
	index Index
}

var _ registry.Registry = &FakeRegistry{}

// NewFakeRegistry creates a new fake registry.
// This implementation should be only used for testing purposes.
// It returns the FakeRegistry itself to be able to add modify functionality in the future
func NewFakeRegistry(refs ...DefinitionReference) *FakeRegistry {
	r := &FakeRegistry{
		index: make(Index),
	}

	for _, obj := range refs {
		ref := obj
		r.index.Add(ref)
	}

	return r
}

// NewFakeRegistryFromPath initializes a FakeRegistry from all found definitions in the given path
func NewFakeRegistryFromPath(path string) (*FakeRegistry, error) {
	var (
		defs    = make([]DefinitionReference, 0)
		decoder = serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder()
	)
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		def := &lsv1alpha1.ComponentDefinition{}
		if _, _, err := decoder.Decode(data, nil, def); err != nil {
			return err
		}
		defs = append(defs, DefinitionReference{
			Definition: *def,
			Fs:         nil,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return NewFakeRegistry(defs...), nil
}

func (f *FakeRegistry) GetDefinitionByRef(ref string) (*lsv1alpha1.ComponentDefinition, error) {
	vn, err := registry.ParseDefinitionRef(ref)
	if err != nil {
		return nil, err
	}
	return f.GetDefinition(vn.Name, vn.Version)
}

func (f *FakeRegistry) GetDefinition(name, version string) (*lsv1alpha1.ComponentDefinition, error) {
	if _, ok := f.index[name]; !ok {
		return nil, registry.NewComponentNotFoundError(name, nil)
	}
	ref, ok := f.index[name][version]
	if !ok {
		return nil, registry.NewVersionNotFoundError(name, version, nil)
	}
	return &ref.Definition, nil
}

func (f *FakeRegistry) GetBlob(name, version string) (afero.Fs, error) {
	if _, ok := f.index[name]; !ok {
		return nil, registry.NewComponentNotFoundError(name, nil)
	}
	ref, ok := f.index[name][version]
	if !ok {
		return nil, registry.NewVersionNotFoundError(name, version, nil)
	}

	return ref.Fs, nil
}

func (f *FakeRegistry) GetVersions(name string) ([]string, error) {
	if _, ok := f.index[name]; !ok {
		return nil, registry.NewComponentNotFoundError(name, nil)
	}
	var (
		versions = make([]string, len(f.index[name]))
		i        = 0
	)
	for version := range f.index[name] {
		versions[i] = version
		i++
	}
	return versions, nil
}

// DefinitionReference is the reference to a fake definition and its filesystem
type DefinitionReference struct {
	Definition lsv1alpha1.ComponentDefinition
	Fs         afero.Fs
}

// Index is internal index structure for definition.
// The map indexes name and secondly the version
/**
name:
	version:
		definition: def
		fs: def
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
