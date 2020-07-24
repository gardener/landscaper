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

package container

import (
	"path"

	corev1 "k8s.io/api/core/v1"
)

// ImportsPathName is the name of the env var that points to the imports file
const ImportsPathName = "IMPORTS_PATH"

// ExportsPathName is the name of the env var that points to the exports file
const ExportsPathName = "EXPORTS_PATH"

// ComponentDescriptorPathName is the name of the env var that points to the component descriptor
const ComponentDescriptorPathName = "COMPONENT_DESCRIPTOR_PATH"

// ContentPathName is the name of the env var that points to the blob content of the definition
const ContentPathName = "CONTENT_PATH"

// OciUserName is the name of the env var that conatins the OCI auth config.
// This env is only set for system containers
const OciConfigName = "OCI_USER"

// BasePath is the base path inside a container that is shared between the main container and ls containers
const BasePath = "/data/ls"

var (
	DefaultEnvVars = []corev1.EnvVar{
		{
			Name:  ImportsPathName,
			Value: path.Join(BasePath, "imports.json"),
		},
		{
			Name:  ExportsPathName,
			Value: path.Join(BasePath, "exports"),
		},
		{
			Name:  ComponentDescriptorPathName,
			Value: path.Join(BasePath, "component_descriptor.json"),
		},
		{
			Name:  ContentPathName,
			Value: path.Join(BasePath, "content"),
		},
	}
)
