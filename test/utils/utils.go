// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"io/ioutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscaper/pkg/apis/core/install"
	corev1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// ReadComponentFromFile reads a file and parses it to a Installation
func ReadComponentFromFile(testfile string) (*corev1alpha1.Installation, error) {
	data, err := ioutil.ReadFile(testfile)
	if err != nil {
		return nil, err
	}

	landscaperScheme := runtime.NewScheme()
	install.Install(landscaperScheme)
	decoder := serializer.NewCodecFactory(landscaperScheme).UniversalDecoder()

	component := &corev1alpha1.Installation{}
	if _, _, err := decoder.Decode(data, nil, component); err != nil {
		return nil, err
	}
	return component, nil
}
