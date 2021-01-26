// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package wait

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
	containeractuator "github.com/gardener/landscaper/pkg/deployer/container"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// UploadExport reads the export config from the given path and stores
// the data as secret in the host cluster
func UploadExport(ctx context.Context, log logr.Logger, kubeClient client.Client, deployItemKey lsv1alpha1.ObjectReference, podKey lsv1alpha1.ObjectReference, exportFilePath string) error {
	pod := &corev1.Pod{}
	if err := kubeClient.Get(ctx, podKey.NamespacedName(), pod); err != nil {
		return err
	}
	mainContainerStatus, err := kutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.MainContainerName)
	if err != nil {
		return err
	}
	// should never happen as we have the wait method before
	if mainContainerStatus.State.Terminated == nil {
		return errors.New("main container not terminated yet")
	}
	if mainContainerStatus.State.Terminated.ExitCode != 0 {
		return fmt.Errorf("main container exists with %d", mainContainerStatus.State.Terminated.ExitCode)
	}

	exportData, err := ioutil.ReadFile(exportFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("no export config found. Skip upload.")
			return nil
		}
		return err
	}

	return createOrUpdateExport(ctx, kubeClient, deployItemKey.Name, podKey.Namespace, exportData)
}

func createOrUpdateExport(ctx context.Context, kubeClient client.Client, deployItemName, namespace string, data []byte) error {
	secret := &corev1.Secret{}
	secret.Name = containeractuator.ExportSecretName(namespace, deployItemName)
	secret.Namespace = namespace

	_, err := kutil.CreateOrUpdate(ctx, kubeClient, secret, func() error {
		kutil.SetMetaDataLabel(&secret.ObjectMeta, container.ContainerDeployerNameLabel, deployItemName)
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: data,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to create or update secret %s in namespace %s: %w", secret.Name, secret.Namespace, err)
	}
	return nil
}
