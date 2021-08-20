//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

SPDX-License-Identifier: Apache-2.0
*/
// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"

	apiscore "github.com/gardener/landscaper/apis/core"
	corev1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	core "github.com/gardener/landscaper/apis/deployer/core"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*Owner)(nil), (*core.Owner)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Owner_To_core_Owner(a.(*Owner), b.(*core.Owner), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*core.Owner)(nil), (*Owner)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_Owner_To_v1alpha1_Owner(a.(*core.Owner), b.(*Owner), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*OwnerList)(nil), (*core.OwnerList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_OwnerList_To_core_OwnerList(a.(*OwnerList), b.(*core.OwnerList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*core.OwnerList)(nil), (*OwnerList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_OwnerList_To_v1alpha1_OwnerList(a.(*core.OwnerList), b.(*OwnerList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*OwnerSpec)(nil), (*core.OwnerSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_OwnerSpec_To_core_OwnerSpec(a.(*OwnerSpec), b.(*core.OwnerSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*core.OwnerSpec)(nil), (*OwnerSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_OwnerSpec_To_v1alpha1_OwnerSpec(a.(*core.OwnerSpec), b.(*OwnerSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*OwnerStatus)(nil), (*core.OwnerStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_OwnerStatus_To_core_OwnerStatus(a.(*OwnerStatus), b.(*core.OwnerStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*core.OwnerStatus)(nil), (*OwnerStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_OwnerStatus_To_v1alpha1_OwnerStatus(a.(*core.OwnerStatus), b.(*OwnerStatus), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_Owner_To_core_Owner(in *Owner, out *core.Owner, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha1_OwnerSpec_To_core_OwnerSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_OwnerStatus_To_core_OwnerStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_Owner_To_core_Owner is an autogenerated conversion function.
func Convert_v1alpha1_Owner_To_core_Owner(in *Owner, out *core.Owner, s conversion.Scope) error {
	return autoConvert_v1alpha1_Owner_To_core_Owner(in, out, s)
}

func autoConvert_core_Owner_To_v1alpha1_Owner(in *core.Owner, out *Owner, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_core_OwnerSpec_To_v1alpha1_OwnerSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_core_OwnerStatus_To_v1alpha1_OwnerStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_core_Owner_To_v1alpha1_Owner is an autogenerated conversion function.
func Convert_core_Owner_To_v1alpha1_Owner(in *core.Owner, out *Owner, s conversion.Scope) error {
	return autoConvert_core_Owner_To_v1alpha1_Owner(in, out, s)
}

func autoConvert_v1alpha1_OwnerList_To_core_OwnerList(in *OwnerList, out *core.OwnerList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]core.Owner)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1alpha1_OwnerList_To_core_OwnerList is an autogenerated conversion function.
func Convert_v1alpha1_OwnerList_To_core_OwnerList(in *OwnerList, out *core.OwnerList, s conversion.Scope) error {
	return autoConvert_v1alpha1_OwnerList_To_core_OwnerList(in, out, s)
}

func autoConvert_core_OwnerList_To_v1alpha1_OwnerList(in *core.OwnerList, out *OwnerList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]Owner)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_core_OwnerList_To_v1alpha1_OwnerList is an autogenerated conversion function.
func Convert_core_OwnerList_To_v1alpha1_OwnerList(in *core.OwnerList, out *OwnerList, s conversion.Scope) error {
	return autoConvert_core_OwnerList_To_v1alpha1_OwnerList(in, out, s)
}

func autoConvert_v1alpha1_OwnerSpec_To_core_OwnerSpec(in *OwnerSpec, out *core.OwnerSpec, s conversion.Scope) error {
	out.Type = in.Type
	out.DeployerId = in.DeployerId
	out.Targets = *(*[]apiscore.ObjectReference)(unsafe.Pointer(&in.Targets))
	return nil
}

// Convert_v1alpha1_OwnerSpec_To_core_OwnerSpec is an autogenerated conversion function.
func Convert_v1alpha1_OwnerSpec_To_core_OwnerSpec(in *OwnerSpec, out *core.OwnerSpec, s conversion.Scope) error {
	return autoConvert_v1alpha1_OwnerSpec_To_core_OwnerSpec(in, out, s)
}

func autoConvert_core_OwnerSpec_To_v1alpha1_OwnerSpec(in *core.OwnerSpec, out *OwnerSpec, s conversion.Scope) error {
	out.Type = in.Type
	out.DeployerId = in.DeployerId
	out.Targets = *(*[]corev1alpha1.ObjectReference)(unsafe.Pointer(&in.Targets))
	return nil
}

// Convert_core_OwnerSpec_To_v1alpha1_OwnerSpec is an autogenerated conversion function.
func Convert_core_OwnerSpec_To_v1alpha1_OwnerSpec(in *core.OwnerSpec, out *OwnerSpec, s conversion.Scope) error {
	return autoConvert_core_OwnerSpec_To_v1alpha1_OwnerSpec(in, out, s)
}

func autoConvert_v1alpha1_OwnerStatus_To_core_OwnerStatus(in *OwnerStatus, out *core.OwnerStatus, s conversion.Scope) error {
	out.Accepted = in.Accepted
	out.ObservedGeneration = in.ObservedGeneration
	return nil
}

// Convert_v1alpha1_OwnerStatus_To_core_OwnerStatus is an autogenerated conversion function.
func Convert_v1alpha1_OwnerStatus_To_core_OwnerStatus(in *OwnerStatus, out *core.OwnerStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_OwnerStatus_To_core_OwnerStatus(in, out, s)
}

func autoConvert_core_OwnerStatus_To_v1alpha1_OwnerStatus(in *core.OwnerStatus, out *OwnerStatus, s conversion.Scope) error {
	out.Accepted = in.Accepted
	out.ObservedGeneration = in.ObservedGeneration
	return nil
}

// Convert_core_OwnerStatus_To_v1alpha1_OwnerStatus is an autogenerated conversion function.
func Convert_core_OwnerStatus_To_v1alpha1_OwnerStatus(in *core.OwnerStatus, out *OwnerStatus, s conversion.Scope) error {
	return autoConvert_core_OwnerStatus_To_v1alpha1_OwnerStatus(in, out, s)
}
