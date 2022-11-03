// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

const (
	LandscaperDomain = "landscaper.gardener.cloud"

	// LandscapeConfigName is the namespace unique name of the landscape configuration
	LandscapeConfigName = "default"

	// DataObjectSecretDataKey is the key of the secret where the landscape and installations stores their merged configuration.
	DataObjectSecretDataKey = "config"

	// LandscaperFinalizer is the finalizer of the landscaper
	LandscaperFinalizer = "finalizer." + LandscaperDomain

	// LandscaperDMFinalizer is the finalizer of the landscaper deployer management.
	LandscaperDMFinalizer = "finalizer.deployermanagement." + LandscaperDomain

	// LandscaperAgentFinalizer is the finalizer of the landscaper agent.
	LandscaperAgentFinalizer = "finalizer.agent." + LandscaperDomain

	// Annotations

	// OperationAnnotation is the annotation that specifies a operation for a component
	OperationAnnotation = LandscaperDomain + "/operation"

	// DeleteWithoutUninstallAnnotation is the annotation that specifies that root installations are deleted without
	// uninstalling the deployed artifacts
	DeleteWithoutUninstallAnnotation = LandscaperDomain + "/delete-without-uninstall"

	// DeleteIgnoreSuccessors is the annotation that specifies that an installation is deleted even if there
	// are dependent installations.
	DeleteIgnoreSuccessors = LandscaperDomain + "/delete-ignore-successors"

	// ReconcileTimestampAnnotation is used to recognize timeouts in deployitems
	ReconcileTimestampAnnotation = LandscaperDomain + "/reconcile-time"

	// AbortTimestampAnnotation is used to recognize timeouts in deployitems
	AbortTimestampAnnotation = LandscaperDomain + "/abort-time"

	// IgnoreAnnotation can be used to stop reconciliation for landscaper resources.
	// Will only have an effect if set to 'true'.
	IgnoreAnnotation = LandscaperDomain + "/ignore"

	// TouchAnnotation can be used to trigger a reconciliation event for a landscaper resource.
	TouchAnnotation = LandscaperDomain + "/touch"

	// RotateTokenAnnotation is the annotation that specifies to rotate a token (used e.g. in the context of TargetSyncObjects)
	RotateTokenAnnotation = LandscaperDomain + "/rotate-token"

	// Labels

	// LandscaperComponentLabelName is the name of the labels the holds the information about landscaper components.
	// This label should be set on landscaper related components like the landscaper controller or deployers.
	LandscaperComponentLabelName = LandscaperDomain + "/component"

	// DeployerRegistrationLabelName is the name of the label that holds the reference to the deployer registration
	// that installation originated from.
	DeployerRegistrationLabelName = "deployers.landscaper.gardener.cloud/deployer-registration"

	// DeployerEnvironmentLabelName is the name of the label that holds the reference to the deployer environment
	// that installation originated from.
	DeployerEnvironmentLabelName = "deployers.landscaper.gardener.cloud/environment"

	// DMEnvironmentTargetAnnotationName is the name of the annotation for the deployer host targets
	// that defines which environment is responsible for the item.
	DMEnvironmentTargetAnnotationName = DeployerEnvironmentLabelName

	// DeployerEnvironmentTargetAnnotationName is the default name for the target selector of specific environments.
	DeployerEnvironmentTargetAnnotationName = LandscaperDomain + "/environment"

	// DeployerOnlyTargetAnnotationName marks a target to be used to deploy only deployers
	DeployerOnlyTargetAnnotationName = LandscaperDomain + "/deployer-only"

	// NotUseDefaultDeployerAnnotation is the installation annotation that refuses the internal deployer to reconcile
	// the installation.
	NotUseDefaultDeployerAnnotation = LandscaperDomain + "/not-internal"

	// Component Descriptor

	// InlineComponentDescriptorLabel is the label name used for nested inline component descriptors
	InlineComponentDescriptorLabel = LandscaperDomain + "/component-descriptor"

	// BlueprintFileName is the filename of a component definition on a local path
	BlueprintFileName = "blueprint.yaml"

	// LandscaperMetricsNamespaceName describes the prometheus metrics namespace for the landscaper component
	LandscaperMetricsNamespaceName = "ociclient"
)

// DeployItem care controller constants
const (
	PickupTimeoutReason      = "PickupTimeout"    // for error messages
	PickupTimeoutOperation   = "WaitingForPickup" // for error messages
	AbortingTimeoutReason    = "AbortingTimeout"  // for error messages
	AbortingTimeoutOperation = "WaitingForAbort"  // for error messages
)
