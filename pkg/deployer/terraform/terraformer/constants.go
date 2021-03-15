package terraformer

import "time"

const (
	// BaseName is the base name used for the terraformer related resources.
	BaseName = "terraformer"
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

	// TerraformStateOutputsKey is the key to retrieve the ouputs from the JSON state.
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
)
