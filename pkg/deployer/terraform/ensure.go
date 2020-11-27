// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/terraformer"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	terraformv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/terraform/v1alpha1"
)

// TerraformFiles are the files that have been rendered from the infrastructure chart.
type TerraformFiles struct {
	Main      string
	Variables string
	TFVars    []byte
}

// Apply runs the equivalent of the terraform apply command.
func (t *Terraform) Apply(ctx context.Context) error {
	currOp := "Apply"
	t.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing

	targetRestConfig, targetClient, err := t.TargetClient()
	if err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "GetTargetClient", err.Error())
		return fmt.Errorf("unable to get target client: %w", err)
	}

	// Create a new Terraformer without authentication.
	var tfr terraformer.Terraformer
	tfr, err = NewTerraformer(targetRestConfig, t.ProviderConfiguration.Namespace, t.DeployItem.Name, t.ProviderConfiguration.TerraformerImage, TerraformerPurpose)
	if err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "InitTerraformer", err.Error())
		return fmt.Errorf("unable to init terraformer: %w", err)
	}
	// Add authentification.
	if len(t.ProviderConfiguration.EnvSecrets) != 0 {
		tfr, err = SetTerraformerEnvVars(ctx, targetClient, tfr, t.ProviderConfiguration)
		if err != nil {
			t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
				currOp, "SetTerraformerAuth", err.Error())
			return fmt.Errorf("unable to set authentification in terraformer: %w", err)
		}
	}

	// Actually apply.
	terraformFiles := t.getTerraformFiles()
	stateInitializer := terraformer.StateConfigMapInitializerFunc(terraformer.CreateState)
	defaultInitializer := terraformer.DefaultInitializer(targetClient, terraformFiles.Main, terraformFiles.Variables, terraformFiles.TFVars, stateInitializer)
	if err := tfr.InitializeWith(defaultInitializer).Apply(); err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "ApplyTerraformConfig", err.Error())
		return fmt.Errorf("failed to apply the terraform config: %w", err)
	}

	// Create status with the terraform outputs.
	var status = &terraformv1alpha1.ProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: terraformv1alpha1.SchemeGroupVersion.String(),
			Kind:       "ProviderStatus",
		},
	}
	output, err := extractOutput(tfr)
	if err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "GetOutputFromTerraformer", err.Error())
		return fmt.Errorf("unable to get terraform output: %w", err)
	}
	status.Output = output
	encStatus, err := encodeProviderStatus(status)
	if err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "EncodeProviderStatus", err.Error())
		return fmt.Errorf("unable to encore provider status: %w", err)
	}

	t.DeployItem.Status.ProviderStatus = encStatus
	t.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	t.DeployItem.Status.ObservedGeneration = t.DeployItem.Generation

	if err := t.kubeClient.Status().Update(ctx, t.DeployItem); err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "UpdateStatus", err.Error())
		return fmt.Errorf("unable to update item status: %w", err)
	}

	t.DeployItem.Status.LastError = nil
	return nil
}

// Destroy runs the equivalent of the terraform destroy command.
func (t *Terraform) Destroy(ctx context.Context) error {
	currOp := "Destroy"
	t.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	targetRestConfig, targetClient, err := t.TargetClient()
	if err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "GetTargetClient", err.Error())
		return fmt.Errorf("unable to get target client: %w", err)
	}

	// Create a new Terraformer without authentication first for house keeping
	// if there is no existing infrastructure to destroy.
	var tfr terraformer.Terraformer
	tfr, err = NewTerraformer(targetRestConfig, t.ProviderConfiguration.Namespace, t.DeployItem.Name, t.ProviderConfiguration.TerraformerImage, TerraformerPurpose)
	if err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "InitTerraformer", err.Error())
		return fmt.Errorf("unable to init terraformer: %w", err)
	}

	// terraform pod from previous reconciliation might still be running,
	// ensure they are gone before doing any operations.
	if err := tfr.EnsureCleanedUp(ctx); err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "CleanupTerraformerPods", err.Error())
		return fmt.Errorf("unable to cleanup orphaned resources: %w", err)
	}

	stateIsEmpty := tfr.IsStateEmpty()
	if stateIsEmpty {
		t.log.Info("infrastructure state is empty, nothing to destroy", currOp, t.DeployItem.Name)
	}

	if !stateIsEmpty {
		// Add authentification to actually destroy infrastructure.
		if len(t.ProviderConfiguration.EnvSecrets) != 0 {
			tfr, err = SetTerraformerEnvVars(ctx, targetClient, tfr, t.ProviderConfiguration)
			if err != nil {
				t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
					currOp, "SetTerraformerAuth", err.Error())
				return fmt.Errorf("unable to set authentification in terraformer: %w", err)
			}
		}

		// Actually destroy.
		if err := tfr.Destroy(); err != nil {
			t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
				currOp, "DestroyTerraformConfig", err.Error())
			return fmt.Errorf("failed to destroy the terraform config")
		}
	}

	// Clean up potentially created configmaps/secrets related to the Terraformer.
	if err := tfr.CleanupConfiguration(ctx); err != nil {
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "CleanupConfiguration", err.Error())
		return fmt.Errorf("failed to clean up configuration: %w", err)
	}

	controllerutil.RemoveFinalizer(t.DeployItem, lsv1alpha1.LandscaperFinalizer)

	return t.kubeClient.Update(ctx, t.DeployItem)
}

// getTerraformFiles returns the terraform files struct from the ProviderConfiguration.
func (t *Terraform) getTerraformFiles() TerraformFiles {
	return TerraformFiles{
		Main:      t.ProviderConfiguration.Main,
		Variables: t.ProviderConfiguration.Variables,
		TFVars:    []byte(t.ProviderConfiguration.TFVars),
	}
}

// extractOutput extracts the outputs from the Terraformer.
func extractOutput(tfr terraformer.Terraformer) (json.RawMessage, error) {
	tfstate, err := tfr.GetState()
	if err != nil {
		return nil, err
	}

	var state map[string]interface{}
	if err := json.Unmarshal(tfstate, &state); err != nil {
		return nil, err
	}

	outputs, ok := state[TerraformStateOutputsKey]
	if !ok {
		return nil, errors.New("no outputs found in the terraform state")
	}

	return json.Marshal(outputs)
}

// encodeProviderStatus encodes a terraform provider status to a RawExtension.
func encodeProviderStatus(status *terraformv1alpha1.ProviderStatus) (*runtime.RawExtension, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: terraformv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return &runtime.RawExtension{}, err
	}

	return raw, nil
}
