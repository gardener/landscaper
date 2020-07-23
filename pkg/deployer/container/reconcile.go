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

package helm

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/gardener/landscaper/pkg/apis/deployer/container"
)

func (c *Container) generatePod() (*corev1.Pod, error) {

	sharedVolume := corev1.Volume{
		Name: "sharedVolume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	sharedVolumeMount := corev1.VolumeMount{
		Name: "sharedData",
		MountPath: container.BasePath,
	}

	initContainer := corev1.Container{
		Name:                     "init",
		Image:                    c.Configuration.InitContainer.Image,
		Command:                  c.Configuration.InitContainer.Command,
		Args:                     c.Configuration.InitContainer.Args,
		Env:                      container.DefaultEnvVars,
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{sharedVolumeMount},
	}

	sidecarContainer := corev1.Container{
		Name:                     "init",
		Image:                    c.Configuration.SidecarContainer.Image,
		Command:                  c.Configuration.SidecarContainer.Command,
		Args:                     c.Configuration.SidecarContainer.Args,
		Env:                      container.DefaultEnvVars,
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{sharedVolumeMount},
	}

	mainContainer := corev1.Container{
		Name:                     "main",
		Image:                    c.ProviderConfiguration.Image,
		Command:                  c.ProviderConfiguration.Command,
		Args:                     c.ProviderConfiguration.Args,
		Env:                      container.DefaultEnvVars,
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{sharedVolumeMount},
	}


	pod := &corev1.Pod{}
	pod.GenerateName = fmt.Sprintf("%s-", c.DeployItem.Name)
	pod.Namespace = c.DeployItem.Namespace
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(300)
	pod.Spec.Volumes = []corev1.Volume{sharedVolume}
	pod.Spec.InitContainers = []corev1.Container{initContainer}
	pod.Spec.Containers = []corev1.Container{mainContainer, sidecarContainer}

	return pod, nil
}
