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

package container

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gardener/landscaper/pkg/apis/deployer/container"
)

// CopyServiceAccountToken copies the container deployer specific token to the
// kubernetes defined location if it exists.
func CopyServiceAccountToken() error {
	if _, err := os.Stat(container.ServiceAccountTokenPath); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}

	// ensure the path exists
	if err := os.MkdirAll(filepath.Dir(PodTokenPath), os.ModePerm); err != nil {
		return err
	}
	in, err := os.Open(container.ServiceAccountTokenPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(PodTokenPath)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
