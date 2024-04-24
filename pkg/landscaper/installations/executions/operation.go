// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions

import (
	"context"
	"fmt"

	lserrors "github.com/gardener/landscaper/apis/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/core/validation"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	genericresolver "github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver/generic"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
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

func (o *ExecutionOperation) RenderDeployItemTemplates(ctx context.Context,
	inst *installations.InstallationImportsAndBlueprint) (core.DeployItemTemplateList, error) {

	op := "RenderDeployItemTemplates"

	cond := lsv1alpha1helper.GetOrInitCondition(inst.GetInstallation().Status.Conditions, lsv1alpha1.ReconcileExecutionCondition)

	templateStateHandler := template.KubernetesStateHandler{
		KubeClient: o.LsUncachedClient(),
		Inst:       inst.GetInstallation(),
	}
	targetResolver := genericresolver.New(o.LsUncachedClient())
	tmpl := template.New(gotemplate.New(templateStateHandler, targetResolver), spiff.New(templateStateHandler, targetResolver))
	executions, err := tmpl.TemplateDeployExecutions(
		template.NewDeployExecutionOptions(
			template.NewBlueprintExecutionOptions(
				o.Context().External.InjectComponentDescriptorRef(inst.GetInstallation()),
				inst.GetBlueprint(),
				o.ComponentVersion,
				o.ResolvedComponentDescriptorList,
				inst.GetImports())))

	if err != nil {
		inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			TemplatingFailedReason, "Unable to template executions"))
		return nil, lserrors.NewWrappedError(err, op, "Template", "unable to template executions", lsv1alpha1.ErrorForInfoOnly)
	}

	if len(executions) == 0 {
		return nil, nil
	}

	// map deployitem specifications into templates for executions
	// includes resolving target import references to target object references
	execTemplates := make(core.DeployItemTemplateList, len(executions))
	for i, elem := range executions {
		var target *core.ObjectReference
		if elem.Target != nil {
			target = &core.ObjectReference{
				Name:      elem.Target.Name,
				Namespace: o.Inst.GetInstallation().Namespace,
			}
			if elem.Target.Index != nil {
				// targetlist import reference
				ti := o.GetTargetListImport(elem.Target.Import)
				if ti == nil {
					return nil, o.deployItemSpecificationError(cond, elem.Name, "targetlist import %q not found", elem.Target.Import)
				}
				if *elem.Target.Index < 0 || *elem.Target.Index >= len(ti.GetTargetExtensions()) {
					return nil, o.deployItemSpecificationError(cond, elem.Name, "index %d out of bounds", *elem.Target.Index)
				}
				rawTarget := ti.GetTargetExtensions()[*elem.Target.Index].GetTarget()
				target.Name = rawTarget.Name
				target.Namespace = rawTarget.Namespace
			} else if elem.Target.Key != nil {
				// targetmap import
				ti := o.GetTargetMapImport(elem.Target.Import)
				if ti == nil {
					return nil, o.deployItemSpecificationError(cond, elem.Name, "targetmap import %q not found", elem.Target.Import)
				}
				targetExt, ok := ti.GetTargetExtensions()[*elem.Target.Key]
				if !ok || targetExt == nil {
					return nil, o.deployItemSpecificationError(cond, elem.Name, "key %q not found in targetmap import %q", *elem.Target.Key, elem.Target.Import)
				}
				rawTarget := targetExt.GetTarget()
				target.Name = rawTarget.Name
				target.Namespace = rawTarget.Namespace
			} else if len(elem.Target.Import) > 0 {
				// single target import reference
				t := o.GetTargetImport(elem.Target.Import)
				if t == nil {
					return nil, o.deployItemSpecificationError(cond, elem.Name, "target import %q not found", elem.Target.Import)
				}
				rawTarget := t.GetTarget()
				target.Name = rawTarget.Name
				target.Namespace = rawTarget.Namespace
			} else if len(elem.Target.Name) == 0 {
				return nil, o.deployItemSpecificationError(cond, elem.Name, "empty target reference")
			}
		}

		// convert timeout
		var timeout *core.Duration
		if elem.Timeout != nil {
			timeout = &core.Duration{Duration: elem.Timeout.Duration}
		}

		execTemplates[i] = core.DeployItemTemplate{
			Name:               elem.Name,
			Type:               elem.Type,
			Target:             target,
			Labels:             elem.Labels,
			Configuration:      elem.Configuration,
			DependsOn:          elem.DependsOn,
			Timeout:            timeout,
			UpdateOnChangeOnly: elem.UpdateOnChangeOnly,
			OnDelete:           elem.OnDelete,
		}
	}

	if err := validation.ValidateDeployItemTemplateList(field.NewPath("deployExecutions"), execTemplates).ToAggregate(); err != nil {
		err2 := fmt.Errorf("error validating deployitem templates: %w", err)
		inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			TemplatingFailedReason, err2.Error()))
		return nil, err2
	}

	return execTemplates, nil
}

func (o *ExecutionOperation) Ensure(ctx context.Context, inst *installations.InstallationImportsAndBlueprint) error {
	execTemplates, err := o.RenderDeployItemTemplates(ctx, inst)
	if execTemplates == nil || err != nil {
		return err
	}

	cond := lsv1alpha1helper.GetOrInitCondition(inst.GetInstallation().Status.Conditions, lsv1alpha1.ReconcileExecutionCondition)

	exec := &lsv1alpha1.Execution{}
	exec.Name = inst.GetInstallation().Name
	exec.Namespace = inst.GetInstallation().Namespace

	versionedDeployItemTemplateList := lsv1alpha1.DeployItemTemplateList{}
	if err := lsv1alpha1.Convert_core_DeployItemTemplateList_To_v1alpha1_DeployItemTemplateList(&execTemplates, &versionedDeployItemTemplateList, nil); err != nil {
		err2 := fmt.Errorf("error converting internal representation of deployitem templates to versioned one: %w", err)
		inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			TemplatingFailedReason, err2.Error()))
		return err2
	}

	if _, err := o.WriterToLsUncachedClient().CreateOrUpdateExecution(ctx, read_write_layer.W000022, exec, func() error {
		exec.Spec.Context = inst.GetInstallation().Spec.Context
		exec.Spec.DeployItems = versionedDeployItemTemplateList

		if lsv1alpha1helper.HasOperation(inst.GetInstallation().ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
			metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ForceReconcileOperation))
		} else {
			metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		}

		lsv1alpha1helper.DeleteCacheHelmChartsAnnotation(&exec.ObjectMeta)
		if lsv1alpha1helper.HasCacheHelmChartsAnnotation(&inst.GetInstallation().ObjectMeta) {
			metav1.SetMetaDataAnnotation(&exec.ObjectMeta, lsv1alpha1.CacheHelmChartsAnnotation, "true")
		}

		if exec.CreationTimestamp.IsZero() && exec.DeletionTimestamp.IsZero() {
			controllerutil.AddFinalizer(exec, lsv1alpha1.LandscaperFinalizer)
		}

		if err := controllerutil.SetControllerReference(inst.GetInstallation(), exec, api.LandscaperScheme); err != nil {
			return err
		}
		o.Scheme().Default(exec)
		return nil
	}); err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			CreateOrUpdateExecutionReason, "Unable to create or update execution")
		_ = o.UpdateInstallationStatus(ctx, inst.GetInstallation(), read_write_layer.W000058, cond)
		return err
	}

	inst.GetInstallation().Status.ExecutionReference = &lsv1alpha1.ObjectReference{
		Name:      exec.Name,
		Namespace: exec.Namespace,
	}
	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		ExecutionDeployedReason, "Deployed execution item")
	if err := o.UpdateInstallationStatus(ctx, inst.GetInstallation(), read_write_layer.W000066, cond); err != nil {
		return err
	}

	return nil
}

// GetExecutionForInstallation returns the execution of an installation.
// The execution can be nil if no execution has been found.
func GetExecutionForInstallation(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) (*lsv1alpha1.Execution, error) {
	exec := &lsv1alpha1.Execution{}
	if err := read_write_layer.GetExecution(ctx, kubeClient, kutil.ObjectKeyFromObject(inst), exec,
		read_write_layer.R000025); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return exec, nil
}

func (o *ExecutionOperation) deployItemSpecificationError(cond lsv1alpha1.Condition, name, message string, args ...interface{}) error {
	err := fmt.Errorf(fmt.Sprintf("invalid deployitem specification %q: ", name)+message, args...)
	o.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
		TemplatingFailedReason, err.Error()))
	return err
}
