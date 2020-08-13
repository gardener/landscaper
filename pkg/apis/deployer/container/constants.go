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
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
)

// ContainerDeployerFinalizer is the finalizer that is set by the container deployer
const ContainerDeployerFinalizer = "finalizer.container.deployer.landscaper.gardener.cloud"

// OperationName is the name of the env var that specifies the current operation that the image should execute
const OperationName = "OPERATION"

// OperationType defines the value of a Operation that is propagated to the container.
type OperationType string

// OperationReconcile is the value of the Operation env var that defines a reconcile operation.
const OperationReconcile OperationType = "RECONCILE"

// OperationDelete is the value of the Operation env var that defines a delete operation.
const OperationDelete OperationType = "DELETE"

// BasePath is the base path inside a container that is shared between the main container and ls containers.
const BasePath = "/data/ls"

// ImportsPathName is the name of the env var that points to the imports file.
const ImportsPathName = "IMPORTS_PATH"

// ImportsPath is the path to the imports file.
var ImportsPath = filepath.Join(BasePath, "imports.json")

// ExportsPathName is the name of the env var that points to the exports file.
const ExportsPathName = "EXPORTS_PATH"

// ExportsPath is the path to the export file.
var ExportsPath = filepath.Join(BasePath, "exports")

// ComponentDescriptorPathName is the name of the env var that points to the component descriptor.
const ComponentDescriptorPathName = "COMPONENT_DESCRIPTOR_PATH"

// ComponentDescriptorPath is the path to the component descriptor file.
var ComponentDescriptorPath = filepath.Join(BasePath, "component_descriptor.json")

// ContentPathName is the name of the env var that points to the blob content of the definition.
const ContentPathName = "CONTENT_PATH"

// ContentPath is the path to the content directory.
var ContentPath = filepath.Join(BasePath, "content")

// StatePathName is the name of the env var that points to the directory where the state can be stored.
const StatePathName = "STATE_PATH"

// StatePath is the path to the state directory.
var StatePath = filepath.Join(BasePath, "state")

// PodName is the name of the env var that contains the name of the pod.
const PodName = "POD_NAME"

// PodNamespaceName is the name of the env var that contains the namespace of the pod.
const PodNamespaceName = "POD_NAMESPACE"

// OciUserName is the name of the env var that contains the OCI auth config.
// This env is only set for system containers
const OciConfigName = "OCI_USER"

// DefinitionReferenceName is the name of the env var that contains the reference to the ComponentDefinition.
const DefinitionReferenceName = "DEFINITION_REFERENCE"

// DeployItemName is the name of the env var that contains name of the source DeployItem.
const DeployItemName = "DEPLOY_ITEM_NAME"

// DeployItemNamespaceName is the name of the env var that contains namespace of the source DeployItem.
const DeployItemNamespaceName = "DEPLOY_ITEM_NAMESPACE"

// MainContainerName is the name of the container running the user workload.
const MainContainerName = "main"

// InitContainerName is the name of the container running the init container.
const InitContainerName = "init"

// SidecarContainerName is the name of the container running the sidecar container.
const SidecarContainerName = "sidecar"

var (
	DefaultEnvVars = []corev1.EnvVar{
		{
			Name:  ImportsPathName,
			Value: ImportsPath,
		},
		{
			Name:  ExportsPathName,
			Value: ExportsPath,
		},
		{
			Name:  ComponentDescriptorPathName,
			Value: ComponentDescriptorPath,
		},
		{
			Name:  ContentPathName,
			Value: ContentPath,
		},
		{
			Name:  StatePathName,
			Value: StatePath,
		},
		{
			Name: PodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: PodNamespaceName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}
)
