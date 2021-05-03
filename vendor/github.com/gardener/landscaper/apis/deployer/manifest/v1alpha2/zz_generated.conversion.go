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
	if err := s.AddGeneratedConversionFunc((*ManagedResourceStatus)(nil), (*manifest.ManagedResourceStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_ManagedResourceStatus_To_manifest_ManagedResourceStatus(a.(*ManagedResourceStatus), b.(*manifest.ManagedResourceStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.ManagedResourceStatus)(nil), (*ManagedResourceStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_ManagedResourceStatus_To_v1alpha2_ManagedResourceStatus(a.(*manifest.ManagedResourceStatus), b.(*ManagedResourceStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Manifest)(nil), (*manifest.Manifest)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_Manifest_To_manifest_Manifest(a.(*Manifest), b.(*manifest.Manifest), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.Manifest)(nil), (*Manifest)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_Manifest_To_v1alpha2_Manifest(a.(*manifest.Manifest), b.(*Manifest), scope)
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
	if err := s.AddGeneratedConversionFunc((*manifest.ProviderStatus)(nil), (*ProviderStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus(a.(*manifest.ProviderStatus), b.(*ProviderStatus), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha2_Configuration_To_manifest_Configuration(in *Configuration, out *manifest.Configuration, s conversion.Scope) error {
	out.Identity = in.Identity
	out.TargetSelector = *(*[]v1alpha1.TargetSelector)(unsafe.Pointer(&in.TargetSelector))
	return nil
}

// Convert_v1alpha2_Configuration_To_manifest_Configuration is an autogenerated conversion function.
func Convert_v1alpha2_Configuration_To_manifest_Configuration(in *Configuration, out *manifest.Configuration, s conversion.Scope) error {
	return autoConvert_v1alpha2_Configuration_To_manifest_Configuration(in, out, s)
}

func autoConvert_manifest_Configuration_To_v1alpha2_Configuration(in *manifest.Configuration, out *Configuration, s conversion.Scope) error {
	out.Identity = in.Identity
	out.TargetSelector = *(*[]v1alpha1.TargetSelector)(unsafe.Pointer(&in.TargetSelector))
	return nil
}

// Convert_manifest_Configuration_To_v1alpha2_Configuration is an autogenerated conversion function.
func Convert_manifest_Configuration_To_v1alpha2_Configuration(in *manifest.Configuration, out *Configuration, s conversion.Scope) error {
	return autoConvert_manifest_Configuration_To_v1alpha2_Configuration(in, out, s)
}

func autoConvert_v1alpha2_HealthChecksConfiguration_To_manifest_HealthChecksConfiguration(in *HealthChecksConfiguration, out *manifest.HealthChecksConfiguration, s conversion.Scope) error {
	out.DisableDefault = in.DisableDefault
	out.Timeout = (*v1alpha1.Duration)(unsafe.Pointer(in.Timeout))
	return nil
}

// Convert_v1alpha2_HealthChecksConfiguration_To_manifest_HealthChecksConfiguration is an autogenerated conversion function.
func Convert_v1alpha2_HealthChecksConfiguration_To_manifest_HealthChecksConfiguration(in *HealthChecksConfiguration, out *manifest.HealthChecksConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha2_HealthChecksConfiguration_To_manifest_HealthChecksConfiguration(in, out, s)
}

func autoConvert_manifest_HealthChecksConfiguration_To_v1alpha2_HealthChecksConfiguration(in *manifest.HealthChecksConfiguration, out *HealthChecksConfiguration, s conversion.Scope) error {
	out.DisableDefault = in.DisableDefault
	out.Timeout = (*v1alpha1.Duration)(unsafe.Pointer(in.Timeout))
	return nil
}

// Convert_manifest_HealthChecksConfiguration_To_v1alpha2_HealthChecksConfiguration is an autogenerated conversion function.
func Convert_manifest_HealthChecksConfiguration_To_v1alpha2_HealthChecksConfiguration(in *manifest.HealthChecksConfiguration, out *HealthChecksConfiguration, s conversion.Scope) error {
	return autoConvert_manifest_HealthChecksConfiguration_To_v1alpha2_HealthChecksConfiguration(in, out, s)
}

func autoConvert_v1alpha2_ManagedResourceStatus_To_manifest_ManagedResourceStatus(in *ManagedResourceStatus, out *manifest.ManagedResourceStatus, s conversion.Scope) error {
	out.Policy = manifest.ManifestPolicy(in.Policy)
	out.Resource = in.Resource
	return nil
}

// Convert_v1alpha2_ManagedResourceStatus_To_manifest_ManagedResourceStatus is an autogenerated conversion function.
func Convert_v1alpha2_ManagedResourceStatus_To_manifest_ManagedResourceStatus(in *ManagedResourceStatus, out *manifest.ManagedResourceStatus, s conversion.Scope) error {
	return autoConvert_v1alpha2_ManagedResourceStatus_To_manifest_ManagedResourceStatus(in, out, s)
}

func autoConvert_manifest_ManagedResourceStatus_To_v1alpha2_ManagedResourceStatus(in *manifest.ManagedResourceStatus, out *ManagedResourceStatus, s conversion.Scope) error {
	out.Policy = ManifestPolicy(in.Policy)
	out.Resource = in.Resource
	return nil
}

// Convert_manifest_ManagedResourceStatus_To_v1alpha2_ManagedResourceStatus is an autogenerated conversion function.
func Convert_manifest_ManagedResourceStatus_To_v1alpha2_ManagedResourceStatus(in *manifest.ManagedResourceStatus, out *ManagedResourceStatus, s conversion.Scope) error {
	return autoConvert_manifest_ManagedResourceStatus_To_v1alpha2_ManagedResourceStatus(in, out, s)
}

func autoConvert_v1alpha2_Manifest_To_manifest_Manifest(in *Manifest, out *manifest.Manifest, s conversion.Scope) error {
	out.Policy = manifest.ManifestPolicy(in.Policy)
	out.Manifest = (*runtime.RawExtension)(unsafe.Pointer(in.Manifest))
	return nil
}

// Convert_v1alpha2_Manifest_To_manifest_Manifest is an autogenerated conversion function.
func Convert_v1alpha2_Manifest_To_manifest_Manifest(in *Manifest, out *manifest.Manifest, s conversion.Scope) error {
	return autoConvert_v1alpha2_Manifest_To_manifest_Manifest(in, out, s)
}

func autoConvert_manifest_Manifest_To_v1alpha2_Manifest(in *manifest.Manifest, out *Manifest, s conversion.Scope) error {
	out.Policy = ManifestPolicy(in.Policy)
	out.Manifest = (*runtime.RawExtension)(unsafe.Pointer(in.Manifest))
	return nil
}

// Convert_manifest_Manifest_To_v1alpha2_Manifest is an autogenerated conversion function.
func Convert_manifest_Manifest_To_v1alpha2_Manifest(in *manifest.Manifest, out *Manifest, s conversion.Scope) error {
	return autoConvert_manifest_Manifest_To_v1alpha2_Manifest(in, out, s)
}

func autoConvert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration(in *ProviderConfiguration, out *manifest.ProviderConfiguration, s conversion.Scope) error {
	out.Kubeconfig = in.Kubeconfig
	out.UpdateStrategy = manifest.UpdateStrategy(in.UpdateStrategy)
	out.HealthChecks = in.HealthChecks
	out.DeleteTimeout = (*core.Duration)(unsafe.Pointer(in.DeleteTimeout))
	out.Manifests = *(*manifest.Manifests)(unsafe.Pointer(&in.Manifests))
	return nil
}

// Convert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration is an autogenerated conversion function.
func Convert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration(in *ProviderConfiguration, out *manifest.ProviderConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha2_ProviderConfiguration_To_manifest_ProviderConfiguration(in, out, s)
}

func autoConvert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(in *manifest.ProviderConfiguration, out *ProviderConfiguration, s conversion.Scope) error {
	out.Kubeconfig = in.Kubeconfig
	out.UpdateStrategy = UpdateStrategy(in.UpdateStrategy)
	out.HealthChecks = in.HealthChecks
	out.DeleteTimeout = (*v1alpha1.Duration)(unsafe.Pointer(in.DeleteTimeout))
	out.Manifests = *(*[]Manifest)(unsafe.Pointer(&in.Manifests))
	return nil
}

// Convert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration is an autogenerated conversion function.
func Convert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(in *manifest.ProviderConfiguration, out *ProviderConfiguration, s conversion.Scope) error {
	return autoConvert_manifest_ProviderConfiguration_To_v1alpha2_ProviderConfiguration(in, out, s)
}

func autoConvert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus(in *ProviderStatus, out *manifest.ProviderStatus, s conversion.Scope) error {
	out.ManagedResources = *(*manifest.ManagedResourceStatusList)(unsafe.Pointer(&in.ManagedResources))
	return nil
}

// Convert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus is an autogenerated conversion function.
func Convert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus(in *ProviderStatus, out *manifest.ProviderStatus, s conversion.Scope) error {
	return autoConvert_v1alpha2_ProviderStatus_To_manifest_ProviderStatus(in, out, s)
}

func autoConvert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus(in *manifest.ProviderStatus, out *ProviderStatus, s conversion.Scope) error {
	out.ManagedResources = *(*ManagedResourceStatusList)(unsafe.Pointer(&in.ManagedResources))
	return nil
}

// Convert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus is an autogenerated conversion function.
func Convert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus(in *manifest.ProviderStatus, out *ProviderStatus, s conversion.Scope) error {
	return autoConvert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus(in, out, s)
}
