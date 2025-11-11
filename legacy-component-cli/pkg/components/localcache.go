// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/constants"
)

// LocalComponentCache implements a components oci cache for local files
type LocalComponentCache struct {
	fs         vfs.FileSystem
	decodeOpts []codec.DecodeOption
}

// NewLocalComponentCache creates a new component cache that caches components on a filesystem.
func NewLocalComponentCache(fs vfs.FileSystem, decodeOpts ...codec.DecodeOption) *LocalComponentCache {
	return &LocalComponentCache{
		fs:         fs,
		decodeOpts: decodeOpts,
	}
}

func (l LocalComponentCache) Get(ctx context.Context, repoCtx cdv2.OCIRegistryRepository, name, version string) (*cdv2.ComponentDescriptor, error) {
	return ResolveInLocalCache(l.fs, repoCtx, name, version, l.decodeOpts...)
}

func (l LocalComponentCache) Store(ctx context.Context, descriptor *cdv2.ComponentDescriptor) error {
	return AddToLocalCache(l.fs, descriptor)
}

var _ cdoci.Cache = &LocalComponentCache{}

// ResolveInLocalCache resolves a component descriptor from a local cache.
// The local cache is expected to have its root at $COMPONENT_REPOSITORY_CACHE_DIR.
// In the root directory each repository context has its own directory, whereas the directory name is $baseurl.replace("/", "-").
// Inside the repository context a component descriptor is cached under $component-name + "-" + $component-version
//
// E.g.
// Given COMPONENT_REPOSITORY_CACHE_DIR="/component-cache";baseUrl="eu.gcr.io/my-context/dev"; component-name="github.com/gardener/landscaper/legacy-component-cli"; component-version="v0.1.0"
// results in the path "/component-cache/eu.gcr.io-my-context-dev/github.com/gardener/landscaper/legacy-component-cli-v0.1.0"
func ResolveInLocalCache(fs vfs.FileSystem, repoCtx cdv2.OCIRegistryRepository, name, version string, decodeOpts ...codec.DecodeOption) (*cdv2.ComponentDescriptor, error) {
	componentPath := LocalCachePath(repoCtx, name, version)

	data, err := vfs.ReadFile(fs, componentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ctf.NotFoundError
		}
		return nil, fmt.Errorf("unable to read file from %q: %w", componentPath, err)
	}
	cd := &cdv2.ComponentDescriptor{}
	if err := codec.Decode(data, cd, decodeOpts...); err != nil {
		return nil, fmt.Errorf("unable to decode component descriptor from %q: %w", componentPath, err)
	}
	return cd, nil
}

// AddToLocalCache stores the given filesystem in the local cache.
// The local cache is expected to have its root at $COMPONENT_REPOSITORY_CACHE_DIR.
// In the root directory each repository context has its own directory, whereas the directory name is $baseurl.replace("/", "-").
// Inside the repository context a component descriptor is cached under $component-name + "-" + $component-version
//
// E.g.
// Given COMPONENT_REPOSITORY_CACHE_DIR="/component-cache";baseUrl="eu.gcr.io/my-context/dev"; component-name="github.com/gardener/landscaper/legacy-component-cli"; component-version="v0.1.0"
// results in the path "/component-cache/eu.gcr.io-my-context-dev/github.com/gardener/landscaper/legacy-component-cli-v0.1.0"
func AddToLocalCache(fs vfs.FileSystem, cd *cdv2.ComponentDescriptor) error {
	repo, err := GetOCIRepositoryContext(cd.GetEffectiveRepositoryContext())
	if err != nil {
		return err
	}
	componentPath := LocalCachePath(repo, cd.GetName(), cd.GetVersion())

	data, err := codec.Encode(cd)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor")
	}
	if err := fs.MkdirAll(filepath.Dir(componentPath), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create components path %q: %w", filepath.Dir(componentPath), err)
	}
	if err := vfs.WriteFile(fs, componentPath, data, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write component to cache at %q: %w", componentPath, err)
	}
	return nil
}

// OCIRef wraps the componentspec provided OciRef function by exposing the cdv2.Repository interface
// that is automatically parsed to a oci registry.
func OCIRef(repository cdv2.Repository, name, version string) (string, error) {
	repoCtx, err := GetOCIRepositoryContext(repository)
	if err != nil {
		return "", err
	}
	return cdoci.OCIRef(repoCtx, name, version)
}

// GetOCIRepositoryContext returns a OCIRegistryRepository from a repository
func GetOCIRepositoryContext(repoCtx cdv2.Repository) (cdv2.OCIRegistryRepository, error) {
	if repoCtx == nil {
		return cdv2.OCIRegistryRepository{}, fmt.Errorf("no repository provided")
	}
	var repo cdv2.OCIRegistryRepository
	switch r := repoCtx.(type) {
	case *cdv2.UnstructuredTypedObject:
		if err := r.DecodeInto(&repo); err != nil {
			return cdv2.OCIRegistryRepository{}, err
		}
	case *cdv2.OCIRegistryRepository:
		repo = *r
	default:
		return cdv2.OCIRegistryRepository{}, fmt.Errorf("unknown repository context type %s", repoCtx.GetType())
	}
	return repo, nil
}

// LocalCachePath returns the filepath for a component defined by its repository context, name and version.
func LocalCachePath(repoCtx cdv2.OCIRegistryRepository, name, version string) string {
	cacheRoot := os.Getenv(constants.ComponentRepositoryCacheDirEnvVar)
	repositoryDir := strings.ReplaceAll(repoCtx.BaseURL, "/", "-")
	return filepath.Join(cacheRoot, repositoryDir, name+"-"+version)
}
