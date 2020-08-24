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

package wait

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/apis/deployer/container"
)

type options struct {
	DefaultBackoff wait.Backoff

	ExportFilePath string
	StatePath      string

	podName      string
	podNamespace string
	PodKey       client.ObjectKey

	deployItemName      string
	deployItemNamespace string
	DeployItemKey       client.ObjectKey
}

// Setup reads necessary options from the expected sources.
func (o *options) Setup() {
	o.ExportFilePath = os.Getenv(container.ExportsPathName)
	o.StatePath = os.Getenv(container.StatePathName)

	o.podName = os.Getenv(container.PodName)
	o.podNamespace = os.Getenv(container.PodNamespaceName)
	o.PodKey = client.ObjectKey{Name: o.podName, Namespace: o.podNamespace}

	o.deployItemName = os.Getenv(container.DeployItemName)
	o.deployItemNamespace = os.Getenv(container.DeployItemNamespaceName)
	o.DeployItemKey = client.ObjectKey{Name: o.deployItemName, Namespace: o.deployItemNamespace}

	// todo: create own backoff method with timeout to gracefully handle timeouts
	o.DefaultBackoff = wait.Backoff{
		Duration: 10 * time.Second,
		Factor:   1.25,
		Steps:    math.MaxInt32, // retry until we are stopped by some timeout
		Cap:      5 * time.Minute,
	}
}

// Validate validates the options data.
func (o *options) Validate() error {
	var err *multierror.Error
	if len(o.ExportFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.ExportsPathName))
	}
	if len(o.StatePath) == 0 {
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
