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

package components

import (
	"context"
	"errors"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *actuator) RunScript(ctx context.Context, namespace string, config *v1alpha1.ScriptConfig) error {
	if config ==  nil {
		return errors.New("config has to be provided")
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Image: config.Image,
					Command: []string{"sh"},
					Args: []string{"-c", config.Script},
					Env: []corev1.EnvVar{
						{
							Name: v1alpha1.ImportConfigEnvVarName,
							Value: v1alpha1.ImportConfigPath,
						},
					},
				},
			},
		},
	}
	if err := a.c.Create(ctx, pod); err != nil {
		return err
	}

	return nil
}
