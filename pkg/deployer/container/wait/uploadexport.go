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
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	containeractuator "github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/kubernetes"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// UploadExport reads the export config from the given path and creates or updates the
// corresponding DeployItem.
// todo: mount restricted kubernetes token to pod
func UploadExport(ctx context.Context, log logr.Logger, kubeClient client.Client, deployItemKey lsv1alpha1.ObjectReference, podKey lsv1alpha1.ObjectReference, exportFilePath string) error {
	pod := &corev1.Pod{}
	if err := kubeClient.Get(ctx, podKey.NamespacedName(), pod); err != nil {
		return err
	}
	mainContainerStatus, err := kubernetesutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.MainContainerName)
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

	deployItem := &lsv1alpha1.DeployItem{}
	if err := kubeClient.Get(ctx, deployItemKey.NamespacedName(), deployItem); err != nil {
		return err
	}

	return createOrUpdateExport(ctx, kubeClient, deployItem, exportData)
}

func createOrUpdateExport(ctx context.Context, kubeClient client.Client, deployItem *lsv1alpha1.DeployItem, data []byte) error {
	secret := &corev1.Secret{}
	secret.Name = containeractuator.ExportSecretName(deployItem)
	secret.Namespace = deployItem.Namespace
	if deployItem.Status.ExportReference != nil {
		if deployItem.Status.ExportReference.Name != secret.Name {
			return fmt.Errorf("expected exported secret with name %s but found %s", secret.Name, deployItem.Status.ExportReference.Name)
		}
	}

	_, err := kubernetesutil.CreateOrUpdate(ctx, kubeClient, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: data,
		}
		return controllerutil.SetControllerReference(deployItem, secret, kubernetes.LandscaperScheme)
	})
	if err != nil {
		return fmt.Errorf("unable to create or update secret %s in namespace %s: %w", secret.Name, secret.Namespace, err)
	}

	deployItem.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	if err := kubeClient.Status().Update(ctx, deployItem); err != nil {
		return err
	}

	return nil
}
