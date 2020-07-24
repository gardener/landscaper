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

	"github.com/go-logr/logr"
	"github.com/docker/cli/cli/config/configfile"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/landscaper/registry/oci"
)

// Init downloads the import config, the component descriptor and the blob content
// to the paths defined by the env vars.
// It also creates all needed directories.
func Init(ctx context.Context, log logr.Logger) error {
	var (
		importsFilePath             = os.Getenv(container.ImportsPathName)
		exportsFilePath             = os.Getenv(container.ExportsPathName)
		componentDescriptorFilePath = os.Getenv(container.ComponentDescriptorPathName)
		contentDirPath              = os.Getenv(container.ContentPathName)

		ociConfig = os.Getenv(container.OciConfigName)
	)

	_, err := createRegistryFromDockerAuthConfig(log, []byte(ociConfig))
	if err != nil {
		return err
	}

	// create all directories
	log.Info("create directories")
	if err := os.MkdirAll(path.Dir(importsFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Dir(exportsFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Dir(componentDescriptorFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(contentDirPath, os.ModePerm); err != nil {
		return err
	}
	log.Info("all directories successfully created")

	log.Info("get component descriptor")

	log.Info("get imports file")

	log.Info("get content blob")

	return nil
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

	reg, err :=oci.New(log, &config.OCIConfiguration{
		ConfigFiles: []string{filepath},
	})
	if err != nil {
		return nil, err
	}

	return reg, nil
}
