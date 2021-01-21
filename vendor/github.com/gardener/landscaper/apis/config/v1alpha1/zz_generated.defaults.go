// +build !ignore_autogenerated

/*
Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

SPDX-License-Identifier: Apache-2.0
*/
// Code generated by defaulter-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&LandscaperConfiguration{}, func(obj interface{}) { SetObjectDefaults_LandscaperConfiguration(obj.(*LandscaperConfiguration)) })
	return nil
}

func SetObjectDefaults_LandscaperConfiguration(in *LandscaperConfiguration) {
	SetDefaults_LandscaperConfiguration(in)
}
