// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package wait

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/util/wait"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
)

type options struct {
	DefaultBackoff wait.Backoff

	ExportFilePath string
	StatePath      string

	podName      string
	podNamespace string
	PodKey       lsv1alpha1.ObjectReference

	deployItemName      string
	deployItemNamespace string
	DeployItemKey       lsv1alpha1.ObjectReference
}

// Setup reads necessary options from the expected sources.
func (o *options) Setup() {
	o.ExportFilePath = os.Getenv(container.ExportsPathName)
	o.StatePath = os.Getenv(container.StatePathName)

	o.podName = os.Getenv(container.PodName)
	o.podNamespace = os.Getenv(container.PodNamespaceName)
	o.PodKey = lsv1alpha1.ObjectReference{Name: o.podName, Namespace: o.podNamespace}

	o.deployItemName = os.Getenv(container.DeployItemName)
	o.deployItemNamespace = os.Getenv(container.DeployItemNamespaceName)
	o.DeployItemKey = lsv1alpha1.ObjectReference{Name: o.deployItemName, Namespace: o.deployItemNamespace}

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
