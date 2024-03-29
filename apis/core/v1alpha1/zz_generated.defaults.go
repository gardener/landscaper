//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by defaulter-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&Blueprint{}, func(obj interface{}) { SetObjectDefaults_Blueprint(obj.(*Blueprint)) })
	scheme.AddTypeDefaultingFunc(&DeployItem{}, func(obj interface{}) { SetObjectDefaults_DeployItem(obj.(*DeployItem)) })
	scheme.AddTypeDefaultingFunc(&DeployItemList{}, func(obj interface{}) { SetObjectDefaults_DeployItemList(obj.(*DeployItemList)) })
	scheme.AddTypeDefaultingFunc(&Execution{}, func(obj interface{}) { SetObjectDefaults_Execution(obj.(*Execution)) })
	scheme.AddTypeDefaultingFunc(&ExecutionList{}, func(obj interface{}) { SetObjectDefaults_ExecutionList(obj.(*ExecutionList)) })
	scheme.AddTypeDefaultingFunc(&Installation{}, func(obj interface{}) { SetObjectDefaults_Installation(obj.(*Installation)) })
	scheme.AddTypeDefaultingFunc(&InstallationList{}, func(obj interface{}) { SetObjectDefaults_InstallationList(obj.(*InstallationList)) })
	return nil
}

func SetObjectDefaults_Blueprint(in *Blueprint) {
	SetDefaults_Blueprint(in)
}

func SetObjectDefaults_DeployItem(in *DeployItem) {
	SetDefaults_DeployItem(in)
}

func SetObjectDefaults_DeployItemList(in *DeployItemList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_DeployItem(a)
	}
}

func SetObjectDefaults_Execution(in *Execution) {
	SetDefaults_Execution(in)
}

func SetObjectDefaults_ExecutionList(in *ExecutionList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Execution(a)
	}
}

func SetObjectDefaults_Installation(in *Installation) {
	SetDefaults_Installation(in)
}

func SetObjectDefaults_InstallationList(in *InstallationList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Installation(a)
	}
}
