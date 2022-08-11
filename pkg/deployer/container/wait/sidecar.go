// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package wait

import (
	"context"
	"io/ioutil"
	"os"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/container/state"
)

// Run runs the container deployer sidecar.
func Run(ctx context.Context) error {
	log, ctx := logging.FromContextOrNew(ctx, nil)
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
			Scheme: api.LandscaperScheme,
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
	if err := WaitUntilMainContainerFinished(ctx, kubeClient, opts.PodKey.NamespacedName()); err != nil {
		return withTerminationLog(log, err)
	}

	// backup state
	if err := state.New(log, kubeClient, opts.podNamespace, opts.DeployItemKey, opts.StatePath).Backup(ctx); err != nil {
		return withTerminationLog(log, err)
	}

	// upload exports
	if err := UploadExport(ctx, kubeClient, opts.DeployItemKey, opts.PodKey, opts.ExportFilePath); err != nil {
		return withTerminationLog(log, err)
	}
	return nil
}

func withTerminationLog(log logging.Logger, err error) error {
	if err == nil {
		return nil
	}

	if err := ioutil.WriteFile("/dev/termination-log", []byte(err.Error()), os.ModePerm); err != nil {
		log.Error(err, "unable to write termination message")
	}
	return err
}
