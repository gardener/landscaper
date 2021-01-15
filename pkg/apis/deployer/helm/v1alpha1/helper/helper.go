// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
)

// ProviderStatusToRawExtension converts a helm status into a raw extension
func ProviderStatusToRawExtension(status *helmv1alpha1.ProviderStatus) (*runtime.RawExtension, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: helmv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return nil, err
	}
	return raw, nil
}

// ProviderConfigurationToRawExtension converts a helm provider configuration into a raw extension
func ProviderConfigurationToRawExtension(config *helmv1alpha1.ProviderConfiguration) (*runtime.RawExtension, error) {
	config.TypeMeta = metav1.TypeMeta{
		APIVersion: helmv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ProviderConfiguration",
	}

	raw := &runtime.RawExtension{}
	obj := config.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return nil, err
	}
	data, err := raw.MarshalJSON()
	if err != nil {
		return nil, err
	}
	raw.Raw = data
	return raw, nil
}
