// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"context"
	"fmt"
	"strings"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"
	secretresolver "github.com/gardener/landscaper/pkg/utils/targetresolver/secret"
)

const (
	// TemplatingFailedReason is the reason that is defined during templating.
	TemplatingFailedReason = "ImportValidationFailed"
)

// ImportOperation templates the executions and handles the interaction with
// the import object.
type ImportOperation struct {
	*installations.Operation
}

// New creates a new executions operations object
func New(op *installations.Operation) *ImportOperation {
	return &ImportOperation{
		Operation: op,
	}
}

func (o *ImportOperation) Ensure(ctx context.Context) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.GetInstallation().Status.Conditions, lsv1alpha1.ValidateImportsCondition)

	templateStateHandler := template.KubernetesStateHandler{
		KubeClient: o.Client(),
		Inst:       o.Inst.GetInstallation(),
	}
	targetResolver := secretresolver.New(o.Client())
	tmpl := template.New(gotemplate.New(o.BlobResolver, templateStateHandler, targetResolver), spiff.New(templateStateHandler))
	errors, bindings, err := tmpl.TemplateImportExecutions(
		template.NewBlueprintExecutionOptions(
			o.Context().External.InjectComponentDescriptorRef(o.Inst.GetInstallation()),
			o.Inst.GetBlueprint(),
			o.ComponentDescriptor,
			o.ResolvedComponentDescriptorList,
			o.Inst.GetImports()))

	if err != nil {
		o.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			TemplatingFailedReason, "Unable to template executions"))
		return fmt.Errorf("unable to template executions: %w", err)
	}

	for k, v := range bindings {
		o.Inst.GetImports()[k] = v
	}
	if len(errors) == 0 {
		return nil
	}

	msg := strings.Join(errors, ", ")
	o.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
		TemplatingFailedReason, msg))
	return fmt.Errorf("import validation failed: %w", fmt.Errorf("%s", msg))
}
