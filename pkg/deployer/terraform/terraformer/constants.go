package terraformer

import (
	"path/filepath"
	"time"
)

const (
	// BaseName is the base name used for the terraformer related resources.
	BaseName = "terraformer"
	// InitContainerName is the name of the init container.
	InitContainerName = "init"
	// ManagedInstanceLabel describes label that is added to every terraform deployer managed resource
	// to define its corresponding instance.
	ManagedInstanceLabel = "terraform.deployer.landscaper.gardener.cloud/instance"
	// ManagedDeployItemLabel describes label that is added to every terraform deployer managed resource
	// to define its source deploy item.
	ManagedDeployItemLabel = "terraform.deployer.landscaper.gardener.cloud/deployitem"
	// LabelKeyName is a key for label on a Terraformer Pod indicating the name of the item.
	LabelKeyItemName = "terraform.deployer.landscaper.gardener.cloud/name"
	// LabelKeyNamespace is a key for label on a Terraformer Pod indicating the namespace of the item.
	LabelKeyItemNamespace = "terraform.deployer.landscaper.gardener.cloud/namespace"
	// LabelKeyGeneration is a key for label on Terraformer Pod indicating the item generation being deployed.
	LabelKeyGeneration = "terraform.deployer.landscaper.gardener.cloud/generation"
	// LabelKeyCommand is a key for label on Terraformer Pod indicating the terraform command being executed.
	LabelKeyCommand = "terraform.deployer.landscaper.gardener.cloud/command"

	// TerraformConfigMainKey is the key of the main configuration in the ConfigurationConfigMap.
	TerraformConfigMainKey = "main.tf"
	// TerraformConfigVarsKey is the key of the variables in the ConfigurationConfigMap.
	TerraformConfigVarsKey = "variables.tf"
	// TerraformTFVarsKey is the key of the TFVars in the VariablesSecret.
	TerraformTFVarsKey = "terraform.tfvars"
	// TerraformStateKey is the key of the state in the StateConfigMap.
	TerraformStateKey = "terraform.tfstate"

	// TerraformConfigSuffix is the suffix used for the ConfigMap which stores the Terraform configuration and variables declaration.
	TerraformConfigSuffix = "tf-config"
	// TerraformTFVarsSuffix is the suffix used for the Secret which stores the Terraform variables definition.
	TerraformTFVarsSuffix = "tf-vars"
	// TerraformStateSuffix is the suffix used for the ConfigMap which stores the Terraform state.
	TerraformStateSuffix = "tf-state"

	// TerraformStateOutputsKey is the key to retrieve the outputs from the JSON state.
	TerraformStateOutputsKey = "outputs"

	// DeadlineCleaning is the deadline while waiting for a clean environment.
	DeadlineCleaning = 5 * time.Minute
	// TerminationGracePeriodSeconds configures the .spec.terminationGracePeriodSeconds for the Terraformer pod.
	TerminationGracePeriodSeconds = 60

	// ApplyCommand it the terraform apply command.
	ApplyCommand = "apply"
	// DestroyCommand it the terraform destroy command.
	DestroyCommand = "destroy"

	// ExitCodeSucceeded is the exit code when the terraformer command succeeded.
	ExitCodeSucceeded int32 = 0

	// TerraformerProvidersPath is the filesystem path to the directory that should contain all providers.
	// This directory is specific to the gardener terraformer implementation.
	// See https://github.com/gardener/terraformer/blob/7db2398aa14f30c056b71f6594954d930b84e009/pkg/terraformer/paths.go#L38
	TerraformerProvidersPath = "/terraform-providers"

	// Env vars

	// DeployItemConfigurationPathName is the name of the env var that points to the provider configuration file.
	DeployItemConfigurationPathName = "CONFIGURATION_PATH"
	// RegistrySecretBasePathName is the environment variable pointing to the file system location of all OCI pull secrets
	RegistrySecretBasePathName = "REGISTRY_SECRETS_DIR"
	// TerraformSharedDirEnvVarName is the name of the environment variable that hold the shared terraform directory
	TerraformSharedDirEnvVarName = "TERRAFORM_SHARED_DIR"
	// TerraformSharedDirEnvVarName is the name of the environment variable that hold the shared terraform providers directory
	TerraformProvidersDirEnvVarName = "TERRAFORM_PROVIDERS_DIR"
)

// BasePath is the base path inside a container that contains the container deployer specific data.
const BasePath = "/data/ls"

// DeployItemConfigurationFilename is the name of the file that contains the provider configuration as json.
const DeployItemConfigurationFilename = "di-configuration.json"

// DeployItemConfigurationPath is the path to the configuration file.
var DeployItemConfigurationPath = filepath.Join(BasePath, "internal", DeployItemConfigurationFilename)

// SharedBasePath is the base path inside the container that is shared between the main and ls containers
var SharedBasePath = filepath.Join(BasePath, "shared")

// RegistrySecretBasePath is the path to all OCI pull secrets
var RegistrySecretBasePath = filepath.Join(BasePath, "registry_secrets")

// SharedProvidersDirectory is the name of the directory where the providers are written to in the shared volume.
const SharedProvidersDirectory = "providers"

// SharedProvidersPath is the path to the directory where the provider is downloaded by the init container.
var SharedProvidersPath = filepath.Join(SharedBasePath, SharedProvidersDirectory)
