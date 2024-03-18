//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by defaulter-gen. DO NOT EDIT.

package manifest

import (
	runtime "k8s.io/apimachinery/pkg/runtime"

	v1alpha1 "github.com/gardener/landscaper/apis/config/v1alpha1"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&Configuration{}, func(obj interface{}) { SetObjectDefaults_Configuration(obj.(*Configuration)) })
	return nil
}

func SetObjectDefaults_Configuration(in *Configuration) {
	v1alpha1.SetDefaults_CommonControllerConfig(&in.Controller.CommonControllerConfig)
}
