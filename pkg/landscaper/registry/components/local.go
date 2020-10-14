// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsregistry

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/go-logr/logr"
)

// LocalRepositoryType defines the local repository context type.
const LocalRepositoryType = "local"

// localClient is a component descriptor repository implementation
// that resolves component references stored locally.
// A ComponentDescriptor is resolved by traversing the given paths and decode every found file as component descriptor.
// todo: build cache to not read every file with every resolve attempt.
type localClient struct {
	log   logr.Logger
	paths []string
}

// NewLocalClient creates a new local registry.
func NewLocalClient(log logr.Logger, paths ...string) (TypedRegistry, error) {
	return &localClient{
		log:   log,
		paths: paths,
	}, nil
}

// Type return the oci registry type that can be handled by this ociClient
func (c *localClient) Type() string {
	return LocalRepositoryType
}

// Get resolves a reference and returns the component descriptor.
func (c *localClient) Resolve(_ context.Context, repoCtx cdv2.RepositoryContext, ref cdv2.ObjectMeta) (*cdv2.ComponentDescriptor, error) {
	if repoCtx.Type != LocalRepositoryType {
		return nil, fmt.Errorf("unsupported type %s expected %s", repoCtx.Type, LocalRepositoryType)
	}

	for _, path := range c.paths {
		cd, err := c.searchInPath(path, ref)
		if err != nil {
			if err != cdv2.NotFound {
				return nil, err
			}
			continue
		}
		return cd, err
	}

	return nil, cdv2.NotFound
}

func (c *localClient) searchInPath(path string, ref cdv2.ObjectMeta) (*cdv2.ComponentDescriptor, error) {
	foundErr := errors.New("comp found")
	var cd *cdv2.ComponentDescriptor
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		// ignore errors
		if err != nil {
			c.log.V(3).Info(err.Error())
			return nil
		}

		if info.IsDir() {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		tmpCD := &cdv2.ComponentDescriptor{}
		if err := codec.Decode(data, tmpCD); err != nil {
			c.log.V(7).Info(fmt.Sprintf("unable to decode file: %s", err.Error()), "file", path)
			return nil
		}

		if tmpCD.GetName() == ref.GetName() && tmpCD.GetVersion() == ref.GetVersion() {
			cd = tmpCD
			return foundErr
		}
		return nil
	})
	if err == nil {
		return nil, cdv2.NotFound
	}
	if err != foundErr {
		return nil, err
	}
	if cd == nil {
		return nil, cdv2.NotFound
	}
	return cd, nil
}
