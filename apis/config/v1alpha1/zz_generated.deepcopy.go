//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

SPDX-License-Identifier: Apache-2.0
*/
// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	corev1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AgentConfiguration) DeepCopyInto(out *AgentConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.OCI != nil {
		in, out := &in.OCI, &out.OCI
		*out = new(OCIConfiguration)
		(*in).DeepCopyInto(*out)
	}
	if in.TargetSelectors != nil {
		in, out := &in.TargetSelectors, &out.TargetSelectors
		*out = make([]corev1alpha1.TargetSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AgentConfiguration.
func (in *AgentConfiguration) DeepCopy() *AgentConfiguration {
	if in == nil {
		return nil
	}
	out := new(AgentConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AgentConfiguration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BlueprintStore) DeepCopyInto(out *BlueprintStore) {
	*out = *in
	in.GarbageCollectionConfiguration.DeepCopyInto(&out.GarbageCollectionConfiguration)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BlueprintStore.
func (in *BlueprintStore) DeepCopy() *BlueprintStore {
	if in == nil {
		return nil
	}
	out := new(BlueprintStore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CommonControllerConfig) DeepCopyInto(out *CommonControllerConfig) {
	*out = *in
	if in.CacheSyncTimeout != nil {
		in, out := &in.CacheSyncTimeout, &out.CacheSyncTimeout
		*out = new(v1.Duration)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CommonControllerConfig.
func (in *CommonControllerConfig) DeepCopy() *CommonControllerConfig {
	if in == nil {
		return nil
	}
	out := new(CommonControllerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContextControllerConfig) DeepCopyInto(out *ContextControllerConfig) {
	*out = *in
	in.Default.DeepCopyInto(&out.Default)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContextControllerConfig.
func (in *ContextControllerConfig) DeepCopy() *ContextControllerConfig {
	if in == nil {
		return nil
	}
	out := new(ContextControllerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContextControllerDefaultConfig) DeepCopyInto(out *ContextControllerDefaultConfig) {
	*out = *in
	if in.ExcludedNamespaces != nil {
		in, out := &in.ExcludedNamespaces, &out.ExcludedNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RepositoryContext != nil {
		in, out := &in.RepositoryContext, &out.RepositoryContext
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContextControllerDefaultConfig.
func (in *ContextControllerDefaultConfig) DeepCopy() *ContextControllerDefaultConfig {
	if in == nil {
		return nil
	}
	out := new(ContextControllerDefaultConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContextsController) DeepCopyInto(out *ContextsController) {
	*out = *in
	in.CommonControllerConfig.DeepCopyInto(&out.CommonControllerConfig)
	in.Config.DeepCopyInto(&out.Config)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContextsController.
func (in *ContextsController) DeepCopy() *ContextsController {
	if in == nil {
		return nil
	}
	out := new(ContextsController)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Controllers) DeepCopyInto(out *Controllers) {
	*out = *in
	if in.SyncPeriod != nil {
		in, out := &in.SyncPeriod, &out.SyncPeriod
		*out = new(v1.Duration)
		**out = **in
	}
	in.Installations.DeepCopyInto(&out.Installations)
	in.Executions.DeepCopyInto(&out.Executions)
	in.DeployItems.DeepCopyInto(&out.DeployItems)
	in.Contexts.DeepCopyInto(&out.Contexts)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Controllers.
func (in *Controllers) DeepCopy() *Controllers {
	if in == nil {
		return nil
	}
	out := new(Controllers)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CrdManagementConfiguration) DeepCopyInto(out *CrdManagementConfiguration) {
	*out = *in
	if in.DeployCustomResourceDefinitions != nil {
		in, out := &in.DeployCustomResourceDefinitions, &out.DeployCustomResourceDefinitions
		*out = new(bool)
		**out = **in
	}
	if in.ForceUpdate != nil {
		in, out := &in.ForceUpdate, &out.ForceUpdate
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CrdManagementConfiguration.
func (in *CrdManagementConfiguration) DeepCopy() *CrdManagementConfiguration {
	if in == nil {
		return nil
	}
	out := new(CrdManagementConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeployItemTimeouts) DeepCopyInto(out *DeployItemTimeouts) {
	*out = *in
	if in.Pickup != nil {
		in, out := &in.Pickup, &out.Pickup
		*out = new(corev1alpha1.Duration)
		**out = **in
	}
	if in.Abort != nil {
		in, out := &in.Abort, &out.Abort
		*out = new(corev1alpha1.Duration)
		**out = **in
	}
	if in.ProgressingDefault != nil {
		in, out := &in.ProgressingDefault, &out.ProgressingDefault
		*out = new(corev1alpha1.Duration)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeployItemTimeouts.
func (in *DeployItemTimeouts) DeepCopy() *DeployItemTimeouts {
	if in == nil {
		return nil
	}
	out := new(DeployItemTimeouts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeployItemsController) DeepCopyInto(out *DeployItemsController) {
	*out = *in
	in.CommonControllerConfig.DeepCopyInto(&out.CommonControllerConfig)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeployItemsController.
func (in *DeployItemsController) DeepCopy() *DeployItemsController {
	if in == nil {
		return nil
	}
	out := new(DeployItemsController)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeployerManagementConfiguration) DeepCopyInto(out *DeployerManagementConfiguration) {
	*out = *in
	in.Agent.DeepCopyInto(&out.Agent)
	if in.DeployerRepositoryContext != nil {
		in, out := &in.DeployerRepositoryContext, &out.DeployerRepositoryContext
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeployerManagementConfiguration.
func (in *DeployerManagementConfiguration) DeepCopy() *DeployerManagementConfiguration {
	if in == nil {
		return nil
	}
	out := new(DeployerManagementConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExecutionsController) DeepCopyInto(out *ExecutionsController) {
	*out = *in
	in.CommonControllerConfig.DeepCopyInto(&out.CommonControllerConfig)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExecutionsController.
func (in *ExecutionsController) DeepCopy() *ExecutionsController {
	if in == nil {
		return nil
	}
	out := new(ExecutionsController)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GarbageCollectionConfiguration) DeepCopyInto(out *GarbageCollectionConfiguration) {
	*out = *in
	if in.ResetInterval != nil {
		in, out := &in.ResetInterval, &out.ResetInterval
		*out = new(v1.Duration)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GarbageCollectionConfiguration.
func (in *GarbageCollectionConfiguration) DeepCopy() *GarbageCollectionConfiguration {
	if in == nil {
		return nil
	}
	out := new(GarbageCollectionConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstallationsController) DeepCopyInto(out *InstallationsController) {
	*out = *in
	in.CommonControllerConfig.DeepCopyInto(&out.CommonControllerConfig)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstallationsController.
func (in *InstallationsController) DeepCopy() *InstallationsController {
	if in == nil {
		return nil
	}
	out := new(InstallationsController)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LandscaperAgentConfiguration) DeepCopyInto(out *LandscaperAgentConfiguration) {
	*out = *in
	in.AgentConfiguration.DeepCopyInto(&out.AgentConfiguration)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LandscaperAgentConfiguration.
func (in *LandscaperAgentConfiguration) DeepCopy() *LandscaperAgentConfiguration {
	if in == nil {
		return nil
	}
	out := new(LandscaperAgentConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LandscaperConfiguration) DeepCopyInto(out *LandscaperConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.Controllers.DeepCopyInto(&out.Controllers)
	if in.RepositoryContext != nil {
		in, out := &in.RepositoryContext, &out.RepositoryContext
		*out = (*in).DeepCopy()
	}
	in.Registry.DeepCopyInto(&out.Registry)
	in.BlueprintStore.DeepCopyInto(&out.BlueprintStore)
	if in.Metrics != nil {
		in, out := &in.Metrics, &out.Metrics
		*out = new(MetricsConfiguration)
		**out = **in
	}
	in.CrdManagement.DeepCopyInto(&out.CrdManagement)
	in.DeployerManagement.DeepCopyInto(&out.DeployerManagement)
	if in.DeployItemTimeouts != nil {
		in, out := &in.DeployItemTimeouts, &out.DeployItemTimeouts
		*out = new(DeployItemTimeouts)
		(*in).DeepCopyInto(*out)
	}
	if in.LsDeployments != nil {
		in, out := &in.LsDeployments, &out.LsDeployments
		*out = new(LsDeployments)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LandscaperConfiguration.
func (in *LandscaperConfiguration) DeepCopy() *LandscaperConfiguration {
	if in == nil {
		return nil
	}
	out := new(LandscaperConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LandscaperConfiguration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LocalRegistryConfiguration) DeepCopyInto(out *LocalRegistryConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LocalRegistryConfiguration.
func (in *LocalRegistryConfiguration) DeepCopy() *LocalRegistryConfiguration {
	if in == nil {
		return nil
	}
	out := new(LocalRegistryConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LsDeployments) DeepCopyInto(out *LsDeployments) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LsDeployments.
func (in *LsDeployments) DeepCopy() *LsDeployments {
	if in == nil {
		return nil
	}
	out := new(LsDeployments)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsConfiguration) DeepCopyInto(out *MetricsConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsConfiguration.
func (in *MetricsConfiguration) DeepCopy() *MetricsConfiguration {
	if in == nil {
		return nil
	}
	out := new(MetricsConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCICacheConfiguration) DeepCopyInto(out *OCICacheConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCICacheConfiguration.
func (in *OCICacheConfiguration) DeepCopy() *OCICacheConfiguration {
	if in == nil {
		return nil
	}
	out := new(OCICacheConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCIConfiguration) DeepCopyInto(out *OCIConfiguration) {
	*out = *in
	if in.ConfigFiles != nil {
		in, out := &in.ConfigFiles, &out.ConfigFiles
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Cache != nil {
		in, out := &in.Cache, &out.Cache
		*out = new(OCICacheConfiguration)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCIConfiguration.
func (in *OCIConfiguration) DeepCopy() *OCIConfiguration {
	if in == nil {
		return nil
	}
	out := new(OCIConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegistryConfiguration) DeepCopyInto(out *RegistryConfiguration) {
	*out = *in
	if in.Local != nil {
		in, out := &in.Local, &out.Local
		*out = new(LocalRegistryConfiguration)
		**out = **in
	}
	if in.OCI != nil {
		in, out := &in.OCI, &out.OCI
		*out = new(OCIConfiguration)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegistryConfiguration.
func (in *RegistryConfiguration) DeepCopy() *RegistryConfiguration {
	if in == nil {
		return nil
	}
	out := new(RegistryConfiguration)
	in.DeepCopyInto(out)
	return out
}
