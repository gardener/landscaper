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

package executions

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/kubernetes"
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

// New creates a new execitions operations object
func New(op *installations.Operation) *ExecutionOperation {
	return &ExecutionOperation{
		Operation: op,
	}
}

func (o *ExecutionOperation) Ensure(ctx context.Context, inst *installations.Installation, imports interface{}) error {
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Info.Status.Conditions, lsv1alpha1.ReconcileExecutionCondition)

	executions, err := template.New(o).TemplateDeployExecutions(inst.Blueprint, o.ResolvedComponentDescriptor, imports)
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
	if _, err := kutil.CreateOrUpdate(ctx, o.Client(), exec, func() error {
		exec.Spec.DeployItems = executions
		metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		if err := controllerutil.SetControllerReference(inst.Info, exec, kubernetes.LandscaperScheme); err != nil {
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
