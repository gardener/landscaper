// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

const (
	// TemplatingFailedReason is the reason that is defined during templating.
	TemplatingFailedReason = "TemplatingFailed"
	// CreateOrUpdateImportsReason is the reason that is defined during
	// the creation or update of the secret containing the imported values
	CreateOrUpdateImportsReason = "CreateOrUpdateImports"
	// CreateOrUpdateExecutionReason is the reason that is defined during the execution create or update.
	CreateOrUpdateExecutionReason = "CreateOrUpdateExecution"
	// ExecutionDeployedReason is the final reason that is defined if the execution is successfully deployed.
	ExecutionDeployedReason = "ExecutionDeployed"
)

// ExecutionOperation templates the executions and handles the interaction with
// the execution object.
type ExecutionOperation struct {
	*installations.Operation
}

// New creates a new executions operations object
func New(op *installations.Operation) *ExecutionOperation {
	return &ExecutionOperation{
		Operation: op,
	}
}

func (o *ExecutionOperation) Ensure(ctx context.Context, inst *installations.Installation) error {
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Info.Status.Conditions, lsv1alpha1.ReconcileExecutionCondition)

	templateStateHandler := template.KubernetesStateHandler{
		KubeClient: o.Client(),
		Inst:       inst.Info,
	}
	tmpl := template.New(gotemplate.New(o.BlobResolver, templateStateHandler), spiff.New(templateStateHandler))
	executions, err := tmpl.TemplateDeployExecutions(template.DeployExecutionOptions{
		Imports:              inst.GetImports(),
		Installation:         o.Context().External.InjectComponentDescriptorRef(inst.Info),
		Blueprint:            inst.Blueprint,
		ComponentDescriptor:  o.ComponentDescriptor,
		ComponentDescriptors: o.ResolvedComponentDescriptorList,
	})
	if err != nil {
		inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			TemplatingFailedReason, "Unable to template executions"))
		return fmt.Errorf("unable to template executions: %w", err)
	}

	if len(executions) == 0 {
		return nil
	}

	exec := &lsv1alpha1.Execution{}
	exec.Name = inst.Info.Name
	exec.Namespace = inst.Info.Namespace
	exec.Spec.RegistryPullSecrets = inst.Info.Spec.RegistryPullSecrets
	if _, err := kutil.CreateOrUpdate(ctx, o.Client(), exec, func() error {
		exec.Spec.DeployItems = executions

		if lsv1alpha1helper.HasOperation(inst.Info.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
			metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ForceReconcileOperation))
		} else {
			metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		}

		if err := controllerutil.SetControllerReference(inst.Info, exec, api.LandscaperScheme); err != nil {
			return err
		}
		return nil
	}); err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			CreateOrUpdateExecutionReason, "Unable to create or update execution")
		_ = o.UpdateInstallationStatus(ctx, inst.Info, lsv1alpha1.ComponentPhaseProgressing, cond)
		return err
	}

	inst.Info.Status.ExecutionReference = &lsv1alpha1.ObjectReference{
		Name:      exec.Name,
		Namespace: exec.Namespace,
	}
	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		ExecutionDeployedReason, "Deployed execution item")
	return o.UpdateInstallationStatus(ctx, inst.Info, inst.Info.Status.Phase, cond)
}

// GetExecutionForInstallation returns the execution of an installation.
// The execution can be nil if no execution has been found.
func GetExecutionForInstallation(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) (*lsv1alpha1.Execution, error) {
	exec := &lsv1alpha1.Execution{}
	if err := kubeClient.Get(ctx, kutil.ObjectKeyFromObject(inst), exec); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return exec, nil
}
