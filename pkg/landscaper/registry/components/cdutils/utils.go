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

package cdutils

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// FindResourceByVersionedReference searches all given components for the defined resource ref.
func FindResourceByVersionedReference(ttype string, ref lsv1alpha1.VersionedResourceReference, components ...cdv2.ComponentDescriptor) (cdv2.Resource, error) {
	for _, comp := range components {
		res, err := FindResourceInComponentByVersionedReference(comp, ttype, ref)
		if !errors.Is(err, cdv2.NotFound) {
			return cdv2.Resource{}, err
		}
		if err == nil {
			return res, nil
		}
	}
	return cdv2.Resource{}, cdv2.NotFound
}

// FindResourceInComponentByVersionedReference searches the given component for the defined resource ref.
func FindResourceInComponentByVersionedReference(comp cdv2.ComponentDescriptor, ttype string, ref lsv1alpha1.VersionedResourceReference) (cdv2.Resource, error) {
	if comp.GetName() != ref.ComponentName {
		return cdv2.Resource{}, cdv2.NotFound
	}
	if comp.GetVersion() != ref.Version {
		return cdv2.Resource{}, cdv2.NotFound
	}

	if ref.Kind != lsv1alpha1.LocalResourceKind && ref.Kind != lsv1alpha1.ExternalResourceKind {
		return cdv2.Resource{}, fmt.Errorf("unexpected resource kind %s: %w", ref.Kind, lsv1alpha1.UnknownResourceKindError)
	}

	if ref.Kind == lsv1alpha1.LocalResourceKind {
		res, err := comp.GetLocalResource(ttype, ref.ResourceName, ref.Version)
		if err != nil {
			return cdv2.Resource{}, err
		}
		return res, nil
	}

	if ref.Kind == lsv1alpha1.ExternalResourceKind {
		res, err := comp.GetExternalResource(ttype, ref.ResourceName, ref.Version)
		if err != nil {
			return cdv2.Resource{}, err
		}
		return res, nil
	}

	return cdv2.Resource{}, cdv2.NotFound
}

// FindResourceByReference searches all given components for the defined resource ref.
func FindResourceByReference(ttype string, ref lsv1alpha1.ResourceReference, components ...cdv2.ComponentDescriptor) (cdv2.Resource, error) {
	for _, comp := range components {
		res, err := FindResourceInComponentByReference(comp, ttype, ref)
		if !errors.Is(err, cdv2.NotFound) {
			return cdv2.Resource{}, err
		}
		if err == nil {
			return res, nil
		}
	}
	return cdv2.Resource{}, cdv2.NotFound
}

// FindResourceInComponentByReference searches the given component for the defined resource ref.
func FindResourceInComponentByReference(comp cdv2.ComponentDescriptor, ttype string, ref lsv1alpha1.ResourceReference) (cdv2.Resource, error) {
	if comp.GetName() != ref.ComponentName {
		return cdv2.Resource{}, cdv2.NotFound
	}

	if ref.Kind != lsv1alpha1.LocalResourceKind && ref.Kind != lsv1alpha1.ExternalResourceKind {
		return cdv2.Resource{}, fmt.Errorf("unexpected resource kind %s: %w", ref.Kind, lsv1alpha1.UnknownResourceKindError)
	}

	if ref.Kind == lsv1alpha1.LocalResourceKind {
		resources := comp.GetLocalResourcesByName(ttype, ref.ResourceName)
		if len(resources) == 0 {
			return cdv2.Resource{}, cdv2.NotFound
		}
		return resources[0], nil
	}

	if ref.Kind == lsv1alpha1.ExternalResourceKind {
		resources := comp.GetExternalResourcesByName(ttype, ref.ResourceName)
		if len(resources) == 0 {
			return cdv2.Resource{}, cdv2.NotFound
		}
		return resources[0], nil
	}

	return cdv2.Resource{}, cdv2.NotFound
}

// BuildComponentDescriptorManifest creates a new manifest from a component descriptor
func BuildComponentDescriptorManifest(cache cache.Cache, cdData []byte) (ocispecv1.Manifest, error) {
	// add dummy json as config to have valid json that is required in a oci manifest
	dummyData := []byte("{}")
	dummyDesc := ocispecv1.Descriptor{
		MediaType: "application/json",
		Digest:    digest.FromBytes(dummyData),
		Size:      int64(len(dummyData)),
	}
	if err := cache.Add(dummyDesc, ioutil.NopCloser(bytes.NewBuffer(dummyData))); err != nil {
		return ocispecv1.Manifest{}, nil
	}

	memFs := memoryfs.New()
	if err := vfs.WriteFile(memFs, filepath.Join("/", componentsregistry.ComponentDescriptorFileName), cdData, os.ModePerm); err != nil {
		return ocispecv1.Manifest{}, err
	}
	var blob bytes.Buffer
	if err := utils.BuildTar(memFs, "/", &blob); err != nil {
		return ocispecv1.Manifest{}, err
	}

	desc := ocispecv1.Descriptor{
		MediaType: componentsregistry.ComponentDescriptorMediaType,
		Digest:    digest.FromBytes(blob.Bytes()),
		Size:      int64(blob.Len()),
	}

	if err := cache.Add(desc, ioutil.NopCloser(&blob)); err != nil {
		return ocispecv1.Manifest{}, err
	}

	manifest := ocispecv1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config:    dummyDesc,
		Layers: []ocispecv1.Descriptor{
			desc,
		},
	}

	return manifest, nil
}

// ComponentReferenceToObjectMeta converts a ComponentReference into a object meta
func ComponentReferenceToObjectMeta(ref cdv2.ComponentReference) cdv2.ObjectMeta {
	return cdv2.ObjectMeta{
		Name:    ref.GetName(),
		Version: ref.GetVersion(),
	}
}

// BuildNewDefinition creates a ocispec Manifest from a component definition.
func BuildNewManifest(cache cache.Cache, data []byte) (*ocispecv1.Manifest, error) {
	memfs := memoryfs.New()
	if err := vfs.WriteFile(memfs, filepath.Join("/", componentsregistry.ComponentDescriptorFileName), data, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to write component descriptor to memory fs: %w", err)
	}

	var blob bytes.Buffer
	if err := utils.BuildTar(memfs, "/", &blob); err != nil {
		return nil, err
	}

	desc := ocispecv1.Descriptor{
		MediaType: componentsregistry.ComponentDescriptorMediaType,
		Digest:    digest.FromBytes(blob.Bytes()),
		Size:      int64(blob.Len()),
	}
	if err := cache.Add(desc, ioutil.NopCloser(&blob)); err != nil {
		return nil, fmt.Errorf("unable to add layer to internal cache: %w", err)
	}

	dummyDesc, err := AddDummyDescriptor(cache)
	if err != nil {
		return nil, fmt.Errorf("unable to add dummy descriptor: %w", err)
	}

	manifest := &ocispecv1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config:    dummyDesc,
		Layers: []ocispecv1.Descriptor{
			desc,
		},
	}

	return manifest, nil
}

// AddDummyDescriptor adds a empty json dummy descriptor.
func AddDummyDescriptor(c cache.Cache) (ocispecv1.Descriptor, error) {
	dummyData := []byte("{}")
	dummyDesc := ocispecv1.Descriptor{
		MediaType: "application/json",
		Digest:    digest.FromBytes(dummyData),
		Size:      int64(len(dummyData)),
	}
	if err := c.Add(dummyDesc, ioutil.NopCloser(bytes.NewBuffer(dummyData))); err != nil {
		return ocispecv1.Descriptor{}, err
	}
	return dummyDesc, nil
}
