//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by deepcopy-gen. DO NOT EDIT.

package managedresource

import (
	runtime "k8s.io/apimachinery/pkg/runtime"

	v1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomResourceGroup) DeepCopyInto(out *CustomResourceGroup) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]ResourceType, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TargetName != nil {
		in, out := &in.TargetName, &out.TargetName
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomResourceGroup.
func (in *CustomResourceGroup) DeepCopy() *CustomResourceGroup {
	if in == nil {
		return nil
	}
	out := new(CustomResourceGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeletionGroupDefinition) DeepCopyInto(out *DeletionGroupDefinition) {
	*out = *in
	if in.PredefinedResourceGroup != nil {
		in, out := &in.PredefinedResourceGroup, &out.PredefinedResourceGroup
		*out = new(PredefinedResourceGroup)
		**out = **in
	}
	if in.CustomResourceGroup != nil {
		in, out := &in.CustomResourceGroup, &out.CustomResourceGroup
		*out = new(CustomResourceGroup)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeletionGroupDefinition.
func (in *DeletionGroupDefinition) DeepCopy() *DeletionGroupDefinition {
	if in == nil {
		return nil
	}
	out := new(DeletionGroupDefinition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Export) DeepCopyInto(out *Export) {
	*out = *in
	if in.FromResource != nil {
		in, out := &in.FromResource, &out.FromResource
		*out = new(v1alpha1.TypedObjectReference)
		**out = **in
	}
	if in.FromObjectReference != nil {
		in, out := &in.FromObjectReference, &out.FromObjectReference
		*out = new(FromObjectReference)
		**out = **in
	}
	if in.TargetName != nil {
		in, out := &in.TargetName, &out.TargetName
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Export.
func (in *Export) DeepCopy() *Export {
	if in == nil {
		return nil
	}
	out := new(Export)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Exports) DeepCopyInto(out *Exports) {
	*out = *in
	if in.Exports != nil {
		in, out := &in.Exports, &out.Exports
		*out = make([]Export, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Exports.
func (in *Exports) DeepCopy() *Exports {
	if in == nil {
		return nil
	}
	out := new(Exports)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FromObjectReference) DeepCopyInto(out *FromObjectReference) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FromObjectReference.
func (in *FromObjectReference) DeepCopy() *FromObjectReference {
	if in == nil {
		return nil
	}
	out := new(FromObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedResourceStatus) DeepCopyInto(out *ManagedResourceStatus) {
	*out = *in
	if in.AnnotateBeforeDelete != nil {
		in, out := &in.AnnotateBeforeDelete, &out.AnnotateBeforeDelete
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PatchBeforeDelete != nil {
		in, out := &in.PatchBeforeDelete, &out.PatchBeforeDelete
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	out.Resource = in.Resource
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedResourceStatus.
func (in *ManagedResourceStatus) DeepCopy() *ManagedResourceStatus {
	if in == nil {
		return nil
	}
	out := new(ManagedResourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ManagedResourceStatusList) DeepCopyInto(out *ManagedResourceStatusList) {
	{
		in := &in
		*out = make(ManagedResourceStatusList, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
		return
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedResourceStatusList.
func (in ManagedResourceStatusList) DeepCopy() ManagedResourceStatusList {
	if in == nil {
		return nil
	}
	out := new(ManagedResourceStatusList)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Manifest) DeepCopyInto(out *Manifest) {
	*out = *in
	if in.Manifest != nil {
		in, out := &in.Manifest, &out.Manifest
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.AnnotateBeforeCreate != nil {
		in, out := &in.AnnotateBeforeCreate, &out.AnnotateBeforeCreate
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.AnnotateBeforeDelete != nil {
		in, out := &in.AnnotateBeforeDelete, &out.AnnotateBeforeDelete
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PatchAfterDeployment != nil {
		in, out := &in.PatchAfterDeployment, &out.PatchAfterDeployment
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.PatchBeforeDelete != nil {
		in, out := &in.PatchBeforeDelete, &out.PatchBeforeDelete
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Manifest.
func (in *Manifest) DeepCopy() *Manifest {
	if in == nil {
		return nil
	}
	out := new(Manifest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PredefinedResourceGroup) DeepCopyInto(out *PredefinedResourceGroup) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PredefinedResourceGroup.
func (in *PredefinedResourceGroup) DeepCopy() *PredefinedResourceGroup {
	if in == nil {
		return nil
	}
	out := new(PredefinedResourceGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceType) DeepCopyInto(out *ResourceType) {
	*out = *in
	if in.Names != nil {
		in, out := &in.Names, &out.Names
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceType.
func (in *ResourceType) DeepCopy() *ResourceType {
	if in == nil {
		return nil
	}
	out := new(ResourceType)
	in.DeepCopyInto(out)
	return out
}
