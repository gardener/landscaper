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
	"fmt"
	"math"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
)

type options struct {
	DefaultBackoff wait.Backoff
	KubeClient     client.Client

	ExportsFilePath             string
	ComponentDescriptorFilePath string
	ContentDirPath              string
	StateDirPath                string

	deployItemName      string
	deployItemNamespace string
	DeployItemKey       client.ObjectKey
	DeployItem          *lsv1alpha1.DeployItem
}

// Complete reads necessary options from the expected sources.
func (o *options) Complete(ctx context.Context) {
	o.ExportsFilePath = os.Getenv(container.ExportsPathName)
	o.ComponentDescriptorFilePath = os.Getenv(container.ComponentDescriptorPathName)
	o.ContentDirPath = os.Getenv(container.ContentPathName)
	o.StateDirPath = os.Getenv(container.StatePathName)

	o.deployItemName = os.Getenv(container.DeployItemName)
	o.deployItemNamespace = os.Getenv(container.DeployItemNamespaceName)
	o.DeployItemKey = client.ObjectKey{Name: o.deployItemName, Namespace: o.deployItemNamespace}

	o.DefaultBackoff = wait.Backoff{
		Duration: 10 * time.Second,
		Factor:   1.25,
		Steps:    math.MaxInt32,
		Cap:      5 * time.Minute,
	}
}

// Validate validates the options data.
func (o *options) Validate() error {
	var err *multierror.Error
	if len(o.ExportsFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.ExportsPathName))
	}
	if len(o.ComponentDescriptorFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.ComponentDescriptorPathName))
	}
	if len(o.ContentDirPath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.ContentPathName))
	}
	if len(o.StateDirPath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.StatePathName))
	}

	if len(o.deployItemName) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.DeployItemName))
	}
	if len(o.deployItemNamespace) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.DeployItemNamespaceName))
	}
	return err.ErrorOrNil()
}
