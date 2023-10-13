// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"unsafe"

	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/landscaper/apis/core"
)

func addConversionFuncs(scheme *runtime.Scheme) error {
	if err := scheme.AddGeneratedConversionFunc((*DeployItemTemplateList)(nil), (*core.DeployItemTemplateList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_DeployItemTemplateList_To_core_DeployItemTemplateList(a.(*DeployItemTemplateList), b.(*core.DeployItemTemplateList), scope)
	}); err != nil {
		return err
	}
	if err := scheme.AddGeneratedConversionFunc((*core.DeployItemTemplateList)(nil), (*DeployItemTemplateList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_DeployItemTemplateList_To_v1alpha1_DeployItemTemplateList(a.(*core.DeployItemTemplateList), b.(*DeployItemTemplateList), scope)
	}); err != nil {
		return err
	}

	if err := scheme.AddGeneratedConversionFunc((*ExecutionSpec)(nil), (*core.ExecutionSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ExecutionSpec_To_core_ExecutionSpec(a.(*ExecutionSpec), b.(*core.ExecutionSpec), scope)
	}); err != nil {
		return err
	}
	if err := scheme.AddGeneratedConversionFunc((*core.ExecutionSpec)(nil), (*ExecutionSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_ExecutionSpec_To_v1alpha1_ExecutionSpec(a.(*core.ExecutionSpec), b.(*ExecutionSpec), scope)
	}); err != nil {
		return err
	}

	return nil
}

// Convert_v1alpha1_ExecutionSpec_To_core_ExecutionSpec is an autogenerated conversion function.
func Convert_v1alpha1_ExecutionSpec_To_core_ExecutionSpec(in *ExecutionSpec, out *core.ExecutionSpec, s conversion.Scope) error {
	if err := Convert_v1alpha1_DeployItemTemplateList_To_core_DeployItemTemplateList((*DeployItemTemplateList)(unsafe.Pointer(&in.DeployItems)), (*core.DeployItemTemplateList)(unsafe.Pointer(&out.DeployItems)), s); err != nil {
		return err
	}
	return nil
}

// Convert_core_ExecutionSpec_To_v1alpha1_ExecutionSpec is an autogenerated conversion function.
func Convert_core_ExecutionSpec_To_v1alpha1_ExecutionSpec(in *core.ExecutionSpec, out *ExecutionSpec, s conversion.Scope) error {
	if err := Convert_core_DeployItemTemplateList_To_v1alpha1_DeployItemTemplateList((*core.DeployItemTemplateList)(unsafe.Pointer(&in.DeployItems)), (*DeployItemTemplateList)(unsafe.Pointer(&out.DeployItems)), s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_DeployItemTemplateList_To_core_DeployItemTemplateList is an autogenerated conversion function.
func Convert_v1alpha1_DeployItemTemplateList_To_core_DeployItemTemplateList(in *DeployItemTemplateList, out *core.DeployItemTemplateList, s conversion.Scope) error {
	if in == nil {
		return nil
	}

	outDeployItemTmplList := make(core.DeployItemTemplateList, len(*in))
	for i, inTmpl := range *in {
		outTmpl := core.DeployItemTemplate{}
		if err := Convert_v1alpha1_DeployItemTemplate_To_core_DeployItemTemplate(&inTmpl, &outTmpl, s); err != nil {
			return err
		}
		outDeployItemTmplList[i] = outTmpl
	}
	*out = outDeployItemTmplList

	return nil
}

// Convert_core_DeployItemTemplateList_To_v1alpha1_DeployItemTemplateList is an autogenerated conversion function.
func Convert_core_DeployItemTemplateList_To_v1alpha1_DeployItemTemplateList(in *core.DeployItemTemplateList, out *DeployItemTemplateList, s conversion.Scope) error {
	if in == nil {
		return nil
	}

	outDeployItemTmplList := make(DeployItemTemplateList, len(*in))
	for i, inTmpl := range *in {
		outTmpl := DeployItemTemplate{}
		if err := Convert_core_DeployItemTemplate_To_v1alpha1_DeployItemTemplate(&inTmpl, &outTmpl, s); err != nil {
			return err
		}
		outDeployItemTmplList[i] = outTmpl
	}
	*out = outDeployItemTmplList

	return nil
}
