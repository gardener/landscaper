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

package cache

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/go-logr/logr"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/afero"
)

type layeredCache struct {
	log logr.Logger
	mux sync.RWMutex

	baseFs    afero.Fs
	overlayFs afero.Fs
}

// NewCache creates a new cache with the given options.
// It uses by default a tmp fs
func NewCache(log logr.Logger, options ...Option) (Cache, error) {
	opts := &Options{}
	opts = opts.ApplyOptions(options)

	if err := initBasePath(opts); err != nil {
		return nil, err
	}

	base := afero.NewBasePathFs(afero.NewOsFs(), opts.BasePath)
	var overlay afero.Fs
	if opts.InMemoryOverlay {
		overlay = afero.NewMemMapFs()
	}

	return &layeredCache{
		log:       log,
		mux:       sync.RWMutex{},
		baseFs:    base,
		overlayFs: overlay,
	}, nil
}

func initBasePath(opts *Options) error {
	if len(opts.BasePath) == 0 {
		path, err := ioutil.TempDir(os.TempDir(), "ocicache")
		if err != nil {
			return err
		}
		opts.BasePath = path
	}
	info, err := os.Stat(opts.BasePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(opts.BasePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if !info.IsDir() {
		return errors.New("path has to be a directory")
	}
	return nil
}

func (lc *layeredCache) Get(desc ocispecv1.Descriptor) (io.ReadCloser, error) {
	path := path(desc)
	lc.mux.RLock()
	defer lc.mux.RUnlock()

	// first search in the overlayFs layer
	if lc.overlayFs != nil {
		if _, err := lc.overlayFs.Stat(path); err == nil {
			return lc.overlayFs.OpenFile(path, os.O_RDONLY, os.ModePerm)
		}
		lc.log.V(7).Info("not found in overlay cache", "path", path, "digest", desc.Digest.Encoded())
	}

	if _, err := lc.baseFs.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	file, err := lc.baseFs.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// copy file to in memory cache
	if lc.overlayFs != nil {
		overlayFile, err := lc.overlayFs.Create(path)
		if err != nil {
			// do not return an error here as we are only unable to write to better cache
			lc.log.V(5).Info(err.Error())
			return file, nil
		}
		defer overlayFile.Close()
		if _, err := io.Copy(overlayFile, file); err != nil {
			// do not return an error here as we are only unable to write to better cache
			lc.log.V(5).Info(err.Error())
			return file, nil
		}
	}
	return file, nil
}

func (lc *layeredCache) Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error {
	path := path(desc)
	lc.mux.Lock()
	defer lc.mux.Unlock()
	defer reader.Close()

	file, err := lc.baseFs.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func path(desc ocispecv1.Descriptor) string {
	return desc.Digest.Encoded()
}
