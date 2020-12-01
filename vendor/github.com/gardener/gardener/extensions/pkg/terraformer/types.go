// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package terraformer

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// terraformer is a struct containing configuration parameters for the Terraform script it acts on.
// * useV2 indicates if it should use flags compatible with terraformer@v2 (defaults to false)
// * purpose is a one-word description depicting what the Terraformer does (e.g. 'infrastructure').
// * namespace is the namespace in which the Terraformer will act.
// * image is the Docker image name of the Terraformer image.
// * configName is the name of the ConfigMap containing the main Terraform file ('main.tf').
// * variablesName is the name of the Secret containing the Terraform variables ('terraform.tfvars').
// * stateName is the name of the ConfigMap containing the Terraform state ('terraform.tfstate').
// * envVars is a list of environment variables which will be injected in the resulting
//   Terraform pod. These variables can contain Terraform variables (i.e., must be prefixed
//   with TF_VAR_).
// * configurationDefined indicates whether the required configuration ConfigMaps/Secrets have been
//   successfully defined.
// * logLevel configures the log level for the Terraformer Pod (only compatible with terraformer@v2,
//   defaults to "info")
// * terminationGracePeriodSeconds is the respective Pod spec field passed to Terraformer Pods.
// * deadlineCleaning is the timeout to wait Terraformer Pods to be cleaned up.
// * deadlinePod is the time to wait apply/destroy Pod to be completed.
type terraformer struct {
	useV2 bool

	logger       logrus.FieldLogger
	client       client.Client
	coreV1Client corev1client.CoreV1Interface

	purpose   string
	name      string
	namespace string
	image     string

	configName           string
	variablesName        string
	stateName            string
	envVars              []corev1.EnvVar
	configurationDefined bool

	logLevel                      string
	terminationGracePeriodSeconds int64

	deadlineCleaning time.Duration
	deadlinePod      time.Duration
}

// RawState represent the terraformer state's raw data
type RawState struct {
	Data     string `json:"data"`
	Encoding string `json:"encoding"`
}

const (
	numberOfConfigResources = 3

	// TerraformerConfigSuffix is the suffix used for the ConfigMap which stores the Terraform configuration and variables declaration.
	TerraformerConfigSuffix = ".tf-config"

	// TerraformerVariablesSuffix is the suffix used for the Secret which stores the Terraform variables definition.
	TerraformerVariablesSuffix = ".tf-vars"

	// TerraformerStateSuffix is the suffix used for the ConfigMap which stores the Terraform state.
	TerraformerStateSuffix = ".tf-state"

	// Base64Encoding denotes base64 encoding for the RawState.Data
	Base64Encoding = "base64"

	// NoneEncoding denotes none encoding for the RawState.Data
	NoneEncoding = "none"
)

// Terraformer is the Terraformer interface.
type Terraformer interface {
	UseV2(bool) Terraformer
	SetLogLevel(string) Terraformer
	// Deprecated: use SetEnvVars instead
	SetVariablesEnvironment(tfVarsEnvironment map[string]string) Terraformer
	SetEnvVars(envVars ...corev1.EnvVar) Terraformer
	SetTerminationGracePeriodSeconds(int64) Terraformer
	SetDeadlineCleaning(time.Duration) Terraformer
	SetDeadlinePod(time.Duration) Terraformer
	InitializeWith(initializer Initializer) Terraformer
	Apply() error
	Destroy() error
	GetRawState(context.Context) (*RawState, error)
	GetState() ([]byte, error)
	IsStateEmpty() bool
	CleanupConfiguration(ctx context.Context) error
	GetStateOutputVariables(variables ...string) (map[string]string, error)
	ConfigExists() (bool, error)
	NumberOfResources(context.Context) (int, error)
	EnsureCleanedUp(ctx context.Context) error
	WaitForCleanEnvironment(ctx context.Context) error
}

// Initializer can initialize a Terraformer.
type Initializer interface {
	Initialize(config *InitializerConfig) error
}

// Factory is a factory that can produce Terraformer and Initializer.
type Factory interface {
	NewForConfig(logger logrus.FieldLogger, config *rest.Config, purpose, namespace, name, image string) (Terraformer, error)
	New(logger logrus.FieldLogger, client client.Client, coreV1Client corev1client.CoreV1Interface, purpose, namespace, name, image string) Terraformer
	DefaultInitializer(c client.Client, main, variables string, tfVars []byte, stateInitializer StateConfigMapInitializer) Initializer
}

// StateConfigMapInitializer initialize terraformer state ConfigMap
type StateConfigMapInitializer interface {
	Initialize(ctx context.Context, c client.Client, namespace, name string) error
}

// StateConfigMapInitializerFunc implements StateConfigMapInitializer
type StateConfigMapInitializerFunc func(ctx context.Context, c client.Client, namespace, name string) error

// CreateOrUpdateState implements StateConfigMapInitializer.
// It use it field state for creating or updating the state ConfigMap
type CreateOrUpdateState struct {
	State *string
}
