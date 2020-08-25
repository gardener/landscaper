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
	"context"
	"io/ioutil"
	"os"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/deployer/container/state"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

// Run runs the container deployer sidecar.
func Run(ctx context.Context, log logr.Logger) error {
	opts := &options{}
	opts.Setup()

	if err := opts.Validate(); err != nil {
		return withTerminationLog(log, err)
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}

	var kubeClient client.Client
	if err := wait.ExponentialBackoff(opts.DefaultBackoff, func() (bool, error) {
		var err error
		kubeClient, err = client.New(restConfig, client.Options{
			Scheme: kubernetes.LandscaperScheme,
		})
		if err != nil {
			log.Error(err, "unable to build kubernetes client")
			return false, nil
		}
		return true, nil
	}); err != nil {
		return err
	}

	// wait for the main container to finish.
	// event if the exitcode != 0, the state is still backed up.
	if err := WaitUntilMainContainerFinished(ctx, log, kubeClient, opts.PodKey.NamespacedName()); err != nil {
		return withTerminationLog(log, err)
	}

	// backup state
	if err := state.New(log, kubeClient, opts.DeployItemKey, opts.StatePath).Backup(ctx); err != nil {
		return withTerminationLog(log, err)
	}

	// upload exports
	if err := UploadExport(ctx, log, kubeClient, opts.DeployItemKey, opts.PodKey, opts.ExportFilePath); err != nil {
		return withTerminationLog(log, err)
	}
	return nil
}

func withTerminationLog(log logr.Logger, err error) error {
	if err == nil {
		return nil
	}

	if err := ioutil.WriteFile("/dev/termination-log", []byte(err.Error()), os.ModePerm); err != nil {
		log.Error(err, "unable to write termination message")
	}
	return err
}
