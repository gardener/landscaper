// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
)

type options struct {
	DefaultBackoff wait.Backoff
	KubeClient     client.Client

	ConfigurationFilePath       string
	ImportsFilePath             string
	ExportsFilePath             string
	ComponentDescriptorFilePath string
	TargetFilePath              string
	ContentDirPath              string
	StateDirPath                string
	RegistrySecretBasePath      string
	OCMConfigFilePath           string

	UseOCM bool

	podNamespace string

	deployItemName      string
	deployItemNamespace string
	DeployItemKey       lsv1alpha1.ObjectReference
	DeployItem          *lsv1alpha1.DeployItem
}

// Complete reads necessary options from the expected sources.
func (o *options) Complete() {
	o.ConfigurationFilePath = os.Getenv(container.ConfigurationPathName)
	o.ImportsFilePath = os.Getenv(container.ImportsPathName)
	o.ExportsFilePath = os.Getenv(container.ExportsPathName)
	o.ComponentDescriptorFilePath = os.Getenv(container.ComponentDescriptorPathName)
	o.TargetFilePath = os.Getenv(container.TargetPathName)
	o.ContentDirPath = os.Getenv(container.ContentPathName)
	o.StateDirPath = os.Getenv(container.StatePathName)
	o.RegistrySecretBasePath = os.Getenv(container.RegistrySecretBasePathName)
	o.OCMConfigFilePath = os.Getenv(container.OCMConfigPathName)

	o.UseOCM = os.Getenv(container.UseOCMName) == "true"

	o.podNamespace = os.Getenv(container.PodNamespaceName)
	o.deployItemName = os.Getenv(container.DeployItemName)
	o.deployItemNamespace = os.Getenv(container.DeployItemNamespaceName)
	o.DeployItemKey = lsv1alpha1.ObjectReference{Name: o.deployItemName, Namespace: o.deployItemNamespace}

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
	if len(o.ConfigurationFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.ConfigurationPathName))
	}
	if len(o.ImportsFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.ImportsPathName))
	}
	if len(o.ExportsFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.ExportsPathName))
	}
	if len(o.ComponentDescriptorFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.ComponentDescriptorPathName))
	}
	if len(o.TargetFilePath) == 0 {
		err = multierror.Append(err, fmt.Errorf("%s has to be defined", container.TargetPathName))
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
