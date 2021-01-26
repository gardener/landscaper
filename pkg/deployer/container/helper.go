// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/version"
)

// DefaultConfiguration sets the defaults for the container deployer configuration.
func DefaultConfiguration(obj *containerv1alpha1.Configuration) {
	if len(obj.Namespace) == 0 {
		obj.Namespace = metav1.NamespaceDefault
	}
	if len(obj.DefaultImage.Image) == 0 {
		obj.DefaultImage.Image = "ubuntu:18.04"
	}
	if len(obj.InitContainer.Image) == 0 {
		obj.InitContainer.Image = fmt.Sprintf("eu.gcr.io/gardener-project/landscaper/container-deployer-init:%s", version.Get().GitVersion)
	}
	if len(obj.WaitContainer.Image) == 0 {
		obj.WaitContainer.Image = fmt.Sprintf("eu.gcr.io/gardener-project/landscaper/container-deployer-wait:%s", version.Get().GitVersion)
	}
}

// DecodeProviderStatus decodes a RawExtension to a container status.
func DecodeProviderStatus(raw *runtime.RawExtension) (*containerv1alpha1.ProviderStatus, error) {
	status := &containerv1alpha1.ProviderStatus{}
	if raw != nil {
		if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(raw.Raw, nil, status); err != nil {
			return nil, err
		}
	}
	return status, nil
}

// EncodeProviderStatus encodes a container status to a RawExtension.
func EncodeProviderStatus(status *containerv1alpha1.ProviderStatus) (*runtime.RawExtension, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: containerv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return &runtime.RawExtension{}, err
	}
	return raw, nil
}
