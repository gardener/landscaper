//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

SPDX-License-Identifier: Apache-2.0
*/
// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha2

import (
	unsafe "unsafe"

	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"

	core "github.com/gardener/landscaper/apis/core"
	v1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifest "github.com/gardener/landscaper/apis/deployer/manifest"
	continuousreconcile "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile"
	managedresource "github.com/gardener/landscaper/apis/deployer/utils/managedresource"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*Configuration)(nil), (*manifest.Configuration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_Configuration_To_manifest_Configuration(a.(*Configuration), b.(*manifest.Configuration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.Configuration)(nil), (*Configuration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_Configuration_To_v1alpha2_Configuration(a.(*manifest.Configuration), b.(*Configuration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Controller)(nil), (*manifest.Controller)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_Controller_To_manifest_Controller(a.(*Controller), b.(*manifest.Controller), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.Controller)(nil), (*Controller)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_Controller_To_v1alpha2_Controller(a.(*manifest.Controller), b.(*Controller), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ExportConfiguration)(nil), (*manifest.ExportConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_ExportConfiguration_To_manifest_ExportConfiguration(a.(*ExportConfiguration), b.(*manifest.ExportConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.ExportConfiguration)(nil), (*ExportConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_ExportConfiguration_To_v1alpha2_ExportConfiguration(a.(*manifest.ExportConfiguration), b.(*ExportConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*HPAConfiguration)(nil), (*manifest.HPAConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_HPAConfiguration_To_manifest_HPAConfiguration(a.(*HPAConfiguration), b.(*manifest.HPAConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.HPAConfiguration)(nil), (*HPAConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_HPAConfiguration_To_v1alpha2_HPAConfiguration(a.(*manifest.HPAConfiguration), b.(*HPAConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ProviderConfiguration)(nil), (*manifest.ProviderConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration(a.(*ProviderConfiguration), b.(*manifest.ProviderConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.ProviderConfiguration)(nil), (*ProviderConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(a.(*manifest.ProviderConfiguration), b.(*ProviderConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ProviderStatus)(nil), (*manifest.ProviderStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus(a.(*ProviderStatus), b.(*manifest.ProviderStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*manifest.ProviderStatus)(nil), (*ProviderStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus(a.(*manifest.ProviderStatus), b.(*ProviderStatus), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha2_Configuration_To_manifest_Configuration(in *Configuration, out *manifest.Configuration, s conversion.Scope) error {
	out.Identity = in.Identity
	out.TargetSelector = *(*[]v1alpha1.TargetSelector)(unsafe.Pointer(&in.TargetSelector))
	if err := Convert_v1alpha2_ExportConfiguration_To_manifest_ExportConfiguration(&in.Export, &out.Export, s); err != nil {
		return err
	}
	out.HPAConfiguration = (*manifest.HPAConfiguration)(unsafe.Pointer(in.HPAConfiguration))
	if err := Convert_v1alpha2_Controller_To_manifest_Controller(&in.Controller, &out.Controller, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha2_Configuration_To_manifest_Configuration is an autogenerated conversion function.
func Convert_v1alpha2_Configuration_To_manifest_Configuration(in *Configuration, out *manifest.Configuration, s conversion.Scope) error {
	return autoConvert_v1alpha2_Configuration_To_manifest_Configuration(in, out, s)
}

func autoConvert_manifest_Configuration_To_v1alpha2_Configuration(in *manifest.Configuration, out *Configuration, s conversion.Scope) error {
	out.Identity = in.Identity
	out.TargetSelector = *(*[]v1alpha1.TargetSelector)(unsafe.Pointer(&in.TargetSelector))
	if err := Convert_manifest_ExportConfiguration_To_v1alpha2_ExportConfiguration(&in.Export, &out.Export, s); err != nil {
		return err
	}
	out.HPAConfiguration = (*HPAConfiguration)(unsafe.Pointer(in.HPAConfiguration))
	if err := Convert_manifest_Controller_To_v1alpha2_Controller(&in.Controller, &out.Controller, s); err != nil {
		return err
	}
	return nil
}

// Convert_manifest_Configuration_To_v1alpha2_Configuration is an autogenerated conversion function.
func Convert_manifest_Configuration_To_v1alpha2_Configuration(in *manifest.Configuration, out *Configuration, s conversion.Scope) error {
	return autoConvert_manifest_Configuration_To_v1alpha2_Configuration(in, out, s)
}

func autoConvert_v1alpha2_Controller_To_manifest_Controller(in *Controller, out *manifest.Controller, s conversion.Scope) error {
	out.CommonControllerConfig = in.CommonControllerConfig
	return nil
}

// Convert_v1alpha2_Controller_To_manifest_Controller is an autogenerated conversion function.
func Convert_v1alpha2_Controller_To_manifest_Controller(in *Controller, out *manifest.Controller, s conversion.Scope) error {
	return autoConvert_v1alpha2_Controller_To_manifest_Controller(in, out, s)
}

func autoConvert_manifest_Controller_To_v1alpha2_Controller(in *manifest.Controller, out *Controller, s conversion.Scope) error {
	out.CommonControllerConfig = in.CommonControllerConfig
	return nil
}

// Convert_manifest_Controller_To_v1alpha2_Controller is an autogenerated conversion function.
func Convert_manifest_Controller_To_v1alpha2_Controller(in *manifest.Controller, out *Controller, s conversion.Scope) error {
	return autoConvert_manifest_Controller_To_v1alpha2_Controller(in, out, s)
}

func autoConvert_v1alpha2_ExportConfiguration_To_manifest_ExportConfiguration(in *ExportConfiguration, out *manifest.ExportConfiguration, s conversion.Scope) error {
	out.DefaultTimeout = (*v1alpha1.Duration)(unsafe.Pointer(in.DefaultTimeout))
	return nil
}

// Convert_v1alpha2_ExportConfiguration_To_manifest_ExportConfiguration is an autogenerated conversion function.
func Convert_v1alpha2_ExportConfiguration_To_manifest_ExportConfiguration(in *ExportConfiguration, out *manifest.ExportConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha2_ExportConfiguration_To_manifest_ExportConfiguration(in, out, s)
}

func autoConvert_manifest_ExportConfiguration_To_v1alpha2_ExportConfiguration(in *manifest.ExportConfiguration, out *ExportConfiguration, s conversion.Scope) error {
	out.DefaultTimeout = (*v1alpha1.Duration)(unsafe.Pointer(in.DefaultTimeout))
	return nil
}

// Convert_manifest_ExportConfiguration_To_v1alpha2_ExportConfiguration is an autogenerated conversion function.
func Convert_manifest_ExportConfiguration_To_v1alpha2_ExportConfiguration(in *manifest.ExportConfiguration, out *ExportConfiguration, s conversion.Scope) error {
	return autoConvert_manifest_ExportConfiguration_To_v1alpha2_ExportConfiguration(in, out, s)
}

func autoConvert_v1alpha2_HPAConfiguration_To_manifest_HPAConfiguration(in *HPAConfiguration, out *manifest.HPAConfiguration, s conversion.Scope) error {
	out.MaxReplicas = in.MaxReplicas
	return nil
}

// Convert_v1alpha2_HPAConfiguration_To_manifest_HPAConfiguration is an autogenerated conversion function.
func Convert_v1alpha2_HPAConfiguration_To_manifest_HPAConfiguration(in *HPAConfiguration, out *manifest.HPAConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha2_HPAConfiguration_To_manifest_HPAConfiguration(in, out, s)
}

func autoConvert_manifest_HPAConfiguration_To_v1alpha2_HPAConfiguration(in *manifest.HPAConfiguration, out *HPAConfiguration, s conversion.Scope) error {
	out.MaxReplicas = in.MaxReplicas
	return nil
}

// Convert_manifest_HPAConfiguration_To_v1alpha2_HPAConfiguration is an autogenerated conversion function.
func Convert_manifest_HPAConfiguration_To_v1alpha2_HPAConfiguration(in *manifest.HPAConfiguration, out *HPAConfiguration, s conversion.Scope) error {
	return autoConvert_manifest_HPAConfiguration_To_v1alpha2_HPAConfiguration(in, out, s)
}

func autoConvert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration(in *ProviderConfiguration, out *manifest.ProviderConfiguration, s conversion.Scope) error {
	out.Kubeconfig = in.Kubeconfig
	out.UpdateStrategy = manifest.UpdateStrategy(in.UpdateStrategy)
	out.ReadinessChecks = in.ReadinessChecks
	out.DeleteTimeout = (*core.Duration)(unsafe.Pointer(in.DeleteTimeout))
	out.Manifests = *(*[]managedresource.Manifest)(unsafe.Pointer(&in.Manifests))
	out.Exports = (*managedresource.Exports)(unsafe.Pointer(in.Exports))
	out.ContinuousReconcile = (*continuousreconcile.ContinuousReconcileSpec)(unsafe.Pointer(in.ContinuousReconcile))
	return nil
}

// Convert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration is an autogenerated conversion function.
func Convert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration(in *ProviderConfiguration, out *manifest.ProviderConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration(in, out, s)
}

func autoConvert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(in *manifest.ProviderConfiguration, out *ProviderConfiguration, s conversion.Scope) error {
	out.Kubeconfig = in.Kubeconfig
	out.UpdateStrategy = UpdateStrategy(in.UpdateStrategy)
	out.ReadinessChecks = in.ReadinessChecks
	out.DeleteTimeout = (*v1alpha1.Duration)(unsafe.Pointer(in.DeleteTimeout))
	out.Manifests = *(*[]managedresource.Manifest)(unsafe.Pointer(&in.Manifests))
	out.Exports = (*managedresource.Exports)(unsafe.Pointer(in.Exports))
	out.ContinuousReconcile = (*continuousreconcile.ContinuousReconcileSpec)(unsafe.Pointer(in.ContinuousReconcile))
	return nil
}

// Convert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration is an autogenerated conversion function.
func Convert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(in *manifest.ProviderConfiguration, out *ProviderConfiguration, s conversion.Scope) error {
	return autoConvert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(in, out, s)
}

func autoConvert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus(in *ProviderStatus, out *manifest.ProviderStatus, s conversion.Scope) error {
	out.ManagedResources = *(*managedresource.ManagedResourceStatusList)(unsafe.Pointer(&in.ManagedResources))
	return nil
}

// Convert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus is an autogenerated conversion function.
func Convert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus(in *ProviderStatus, out *manifest.ProviderStatus, s conversion.Scope) error {
	return autoConvert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus(in, out, s)
}

func autoConvert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus(in *manifest.ProviderStatus, out *ProviderStatus, s conversion.Scope) error {
	out.ManagedResources = *(*managedresource.ManagedResourceStatusList)(unsafe.Pointer(&in.ManagedResources))
	// WARNING: in.AnnotateBeforeCreate requires manual conversion: does not exist in peer-type
	// WARNING: in.AnnotateBeforeDelete requires manual conversion: does not exist in peer-type
	return nil
}
