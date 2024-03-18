//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by deepcopy-gen. DO NOT EDIT.

package container

import (
	json "encoding/json"

	runtime "k8s.io/apimachinery/pkg/runtime"

	config "github.com/gardener/landscaper/apis/config"
	v1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	continuousreconcile "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Configuration) DeepCopyInto(out *Configuration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.OCI != nil {
		in, out := &in.OCI, &out.OCI
		*out = new(config.OCIConfiguration)
		(*in).DeepCopyInto(*out)
	}
	if in.TargetSelector != nil {
		in, out := &in.TargetSelector, &out.TargetSelector
		*out = make([]v1alpha1.TargetSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.DefaultImage.DeepCopyInto(&out.DefaultImage)
	in.InitContainer.DeepCopyInto(&out.InitContainer)
	in.WaitContainer.DeepCopyInto(&out.WaitContainer)
	out.GarbageCollection = in.GarbageCollection
	if in.DebugOptions != nil {
		in, out := &in.DebugOptions, &out.DebugOptions
		*out = new(DebugOptions)
		**out = **in
	}
	if in.HPAConfiguration != nil {
		in, out := &in.HPAConfiguration, &out.HPAConfiguration
		*out = new(HPAConfiguration)
		**out = **in
	}
	in.Controller.DeepCopyInto(&out.Controller)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Configuration.
func (in *Configuration) DeepCopy() *Configuration {
	if in == nil {
		return nil
	}
	out := new(Configuration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Configuration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerSpec) DeepCopyInto(out *ContainerSpec) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerSpec.
func (in *ContainerSpec) DeepCopy() *ContainerSpec {
	if in == nil {
		return nil
	}
	out := new(ContainerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerStatus) DeepCopyInto(out *ContainerStatus) {
	*out = *in
	in.State.DeepCopyInto(&out.State)
	if in.ExitCode != nil {
		in, out := &in.ExitCode, &out.ExitCode
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerStatus.
func (in *ContainerStatus) DeepCopy() *ContainerStatus {
	if in == nil {
		return nil
	}
	out := new(ContainerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Controller) DeepCopyInto(out *Controller) {
	*out = *in
	in.CommonControllerConfig.DeepCopyInto(&out.CommonControllerConfig)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Controller.
func (in *Controller) DeepCopy() *Controller {
	if in == nil {
		return nil
	}
	out := new(Controller)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DebugOptions) DeepCopyInto(out *DebugOptions) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DebugOptions.
func (in *DebugOptions) DeepCopy() *DebugOptions {
	if in == nil {
		return nil
	}
	out := new(DebugOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GarbageCollection) DeepCopyInto(out *GarbageCollection) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GarbageCollection.
func (in *GarbageCollection) DeepCopy() *GarbageCollection {
	if in == nil {
		return nil
	}
	out := new(GarbageCollection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HPAConfiguration) DeepCopyInto(out *HPAConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HPAConfiguration.
func (in *HPAConfiguration) DeepCopy() *HPAConfiguration {
	if in == nil {
		return nil
	}
	out := new(HPAConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodStatus) DeepCopyInto(out *PodStatus) {
	*out = *in
	if in.LastRun != nil {
		in, out := &in.LastRun, &out.LastRun
		*out = (*in).DeepCopy()
	}
	if in.LastSuccessfulJobID != nil {
		in, out := &in.LastSuccessfulJobID, &out.LastSuccessfulJobID
		*out = new(string)
		**out = **in
	}
	in.ContainerStatus.DeepCopyInto(&out.ContainerStatus)
	in.InitContainerStatus.DeepCopyInto(&out.InitContainerStatus)
	in.WaitContainerStatus.DeepCopyInto(&out.WaitContainerStatus)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodStatus.
func (in *PodStatus) DeepCopy() *PodStatus {
	if in == nil {
		return nil
	}
	out := new(PodStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProviderConfiguration) DeepCopyInto(out *ProviderConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ImportValues != nil {
		in, out := &in.ImportValues, &out.ImportValues
		*out = make(json.RawMessage, len(*in))
		copy(*out, *in)
	}
	if in.Blueprint != nil {
		in, out := &in.Blueprint, &out.Blueprint
		*out = new(v1alpha1.BlueprintDefinition)
		(*in).DeepCopyInto(*out)
	}
	if in.ComponentDescriptor != nil {
		in, out := &in.ComponentDescriptor, &out.ComponentDescriptor
		*out = new(v1alpha1.ComponentDescriptorDefinition)
		(*in).DeepCopyInto(*out)
	}
	if in.RegistryPullSecrets != nil {
		in, out := &in.RegistryPullSecrets, &out.RegistryPullSecrets
		*out = make([]v1alpha1.ObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.ContinuousReconcile != nil {
		in, out := &in.ContinuousReconcile, &out.ContinuousReconcile
		*out = new(continuousreconcile.ContinuousReconcileSpec)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProviderConfiguration.
func (in *ProviderConfiguration) DeepCopy() *ProviderConfiguration {
	if in == nil {
		return nil
	}
	out := new(ProviderConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProviderConfiguration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProviderStatus) DeepCopyInto(out *ProviderStatus) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.PodStatus != nil {
		in, out := &in.PodStatus, &out.PodStatus
		*out = new(PodStatus)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProviderStatus.
func (in *ProviderStatus) DeepCopy() *ProviderStatus {
	if in == nil {
		return nil
	}
	out := new(ProviderStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProviderStatus) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
