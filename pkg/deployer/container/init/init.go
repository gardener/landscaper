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

package init

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/afero"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/landscaper/registry/oci"
)

// Run downloads the import config, the component descriptor and the blob content
// to the paths defined by the env vars.
// It also creates all needed directories.
func Run(ctx context.Context, log logr.Logger) error {
	var (
		exportsFilePath             = os.Getenv(container.ExportsPathName)
		componentDescriptorFilePath = os.Getenv(container.ComponentDescriptorPathName)
		contentDirPath              = os.Getenv(container.ContentPathName)
		stateDirPath                = os.Getenv(container.StatePathName)

		defRef    = os.Getenv(container.DefinitionReferenceName)
		ociConfig = os.Getenv(container.OciConfigName)
	)

	reg, err := createRegistryFromDockerAuthConfig(log, []byte(ociConfig))
	if err != nil {
		return err
	}

	// create all directories
	log.Info("create directories")
	if err := os.MkdirAll(path.Dir(exportsFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Dir(componentDescriptorFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(contentDirPath, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(stateDirPath, os.ModePerm); err != nil {
		return err
	}
	log.Info("all directories successfully created")

	log.Info("get component descriptor")

	log.Info("get content blob")
	blobFS, err := reg.GetContent(ctx, nil) // todo: read reference from component descriptor
	if err != nil {
		return err
	}

	osFS := afero.NewOsFs()
	if err := copyFS(blobFS, osFS, "/", contentDirPath); err != nil {
		return err
	}

	log.Info("get state")

	return nil
}

func copyFS(src, dst afero.Fs, srcPath, dstPath string) error {
	return afero.Walk(src, srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dstFilePath := filepath.Join(dstPath, path)
		if info.IsDir() {
			if err := dst.MkdirAll(dstFilePath, info.Mode()); err != nil {
				return err
			}
			return nil
		}

		file, err := src.OpenFile(path, os.O_RDONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		return afero.WriteReader(dst, dstFilePath, file)
	})
}

func createRegistryFromDockerAuthConfig(log logr.Logger, configData []byte) (registry.Registry, error) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "oci-auth-")
	if err != nil {
		return nil, err
	}
	defer tmpfile.Close()
	filepath := path.Join(os.TempDir(), tmpfile.Name())

	if _, err := io.Copy(tmpfile, bytes.NewBuffer(configData)); err != nil {
		return nil, err
	}

	reg, err := oci.New(log, &config.OCIConfiguration{
		ConfigFiles: []string{filepath},
	})
	if err != nil {
		return nil, err
	}

	return reg, nil
}
