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
			c.log.V(3).Info(err.Error())
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
