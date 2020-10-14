// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

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
