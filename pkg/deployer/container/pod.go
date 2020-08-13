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
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
)

// PodOptions contains the configuration that is needed for the scheduled pod
type PodOptions struct {
	ProviderConfiguration *containerv1alpha1.ProviderConfiguration
	InitContainer         containerv1alpha1.ContainerSpec
	SidecarContainer      containerv1alpha1.ContainerSpec

	Name      string
	Namespace string

	Operation     container.OperationType
	DefinitionRef string
	ImportsRef    lsv1alpha1.ObjectReference
	OCIConfig     []byte
}

func generatePod(opts PodOptions) (*corev1.Pod, error) {

	sharedVolume := corev1.Volume{
		Name: "sharedVolume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	sharedVolumeMount := corev1.VolumeMount{
		Name:      "sharedVolume",
		MountPath: container.BasePath,
	}

	importsVolume := corev1.Volume{
		Name: "imports",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: opts.ImportsRef.Name,
				Items: []corev1.KeyToPath{
					{
						Key:  lsv1alpha1.DataObjectSecretDataKey,
						Path: filepath.Base(container.ImportsPath),
					},
				},
			},
		},
	}
	importsVolumeMount := corev1.VolumeMount{
		Name:      "imports",
		MountPath: container.BasePath,
		SubPath:   filepath.Base(container.ImportsPath),
	}

	additionalInitEnvVars := []corev1.EnvVar{
		{
			Name:  container.DefinitionReferenceName,
			Value: opts.DefinitionRef,
		},
		{
			Name:  container.OciConfigName,
			Value: string(opts.OCIConfig),
		},
	}
	additionalSidecarEnvVars := []corev1.EnvVar{
		{
			Name:  container.DefinitionReferenceName,
			Value: opts.DefinitionRef,
		},
		{
			Name:  container.OciConfigName,
			Value: string(opts.OCIConfig),
		},
		{
			Name:  container.DeployItemNamespaceName,
			Value: opts.Namespace,
		},
	}
	additionalEnvVars := []corev1.EnvVar{
		{
			Name:  container.OperationName,
			Value: string(opts.Operation),
		},
		{
			Name:  container.DefinitionReferenceName,
			Value: opts.DefinitionRef,
		},
		{
			Name:  container.DefinitionReferenceName,
			Value: opts.DefinitionRef,
		},
	}

	volumes := []corev1.Volume{
		sharedVolume,
		importsVolume,
	}

	initContainer := corev1.Container{
		Name:                     container.InitContainerName,
		Image:                    opts.InitContainer.Image,
		Command:                  opts.InitContainer.Command,
		Args:                     opts.InitContainer.Args,
		Env:                      append(container.DefaultEnvVars, additionalInitEnvVars...),
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		VolumeMounts:             []corev1.VolumeMount{sharedVolumeMount},
	}

	sidecarContainer := corev1.Container{
		Name:                     container.SidecarContainerName,
		Image:                    opts.SidecarContainer.Image,
		Command:                  opts.SidecarContainer.Command,
		Args:                     opts.SidecarContainer.Args,
		Env:                      append(container.DefaultEnvVars, additionalSidecarEnvVars...),
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{
			sharedVolumeMount,
			importsVolumeMount,
		},
	}

	mainContainer := corev1.Container{
		Name:                     container.MainContainerName,
		Image:                    opts.ProviderConfiguration.Image,
		Command:                  opts.ProviderConfiguration.Command,
		Args:                     opts.ProviderConfiguration.Args,
		Env:                      append(container.DefaultEnvVars, additionalEnvVars...),
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		VolumeMounts:             []corev1.VolumeMount{sharedVolumeMount},
	}

	pod := &corev1.Pod{}
	pod.GenerateName = opts.Name + "-"
	pod.Namespace = opts.Namespace
	pod.Finalizers = []string{container.ContainerDeployerFinalizer}

	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(300)
	pod.Spec.Volumes = volumes
	pod.Spec.InitContainers = []corev1.Container{initContainer}
	pod.Spec.Containers = []corev1.Container{mainContainer, sidecarContainer}

	return pod, nil
}
