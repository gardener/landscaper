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
	"context"
	"os"
	"path"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/pkg/apis/deployer/container"
)

// Init downloads the import config, the component descriptor and the blob content
// to the paths defined by the env vars.
// It also creates all needed directories.
func Init(ctx context.Context, log logr.Logger) error {
	importsFilePath := os.Getenv(container.ImportsPathName)
	exportsFilePath := os.Getenv(container.ExportsPathName)
	componentDescriptorFilePath := os.Getenv(container.ComponentDescriptorPathName)
	contentDirPath := os.Getenv(container.ContentPathName)


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

	log.Info("download imports file")
	return nil
}
