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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/Masterminds/sprig"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
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
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Info.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)

	executions, err := o.template(inst.Blueprint, imports)
	if err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"TemplatingFailed", "Unable to template executions")
		_ = o.UpdateInstallationStatus(ctx, inst.Info, lsv1alpha1.ComponentPhaseProgressing, cond)
		return fmt.Errorf("unable to template executions: %w", err)
	}

	if len(executions) == 0 {
		return nil
	}

	exec := &lsv1alpha1.Execution{}
	exec.Name = inst.Info.Name
	exec.Namespace = inst.Info.Namespace

	if _, err := kubernetesutil.CreateOrUpdate(ctx, o.Client(), exec, func() error {
		exec.Spec.BlueprintRef = &inst.Info.Spec.BlueprintRef
		exec.Spec.RegistryPullSecrets = inst.Info.Spec.RegistryPullSecrets
		exec.Spec.ImportReference = inst.Info.Status.ImportReference
		exec.Spec.Executions = executions
		if err := controllerutil.SetControllerReference(inst.Info, exec, kubernetes.LandscaperScheme); err != nil {
			return err
		}
		return nil
	}); err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"CreateOrUpdateExecution", "Unable to create or update execution")
		_ = o.UpdateInstallationStatus(ctx, inst.Info, lsv1alpha1.ComponentPhaseProgressing, cond)
		return err
	}

	inst.Info.Status.ExecutionReference = &lsv1alpha1.ObjectReference{
		Name:      exec.Name,
		Namespace: exec.Namespace,
	}
	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"ExecutionDeployed", "Deployed execution item")
	return o.UpdateInstallationStatus(ctx, inst.Info, inst.Info.Status.Phase, cond)
}

func (o *ExecutionOperation) template(blueprint *blueprints.Blueprint, imports interface{}) ([]lsv1alpha1.ExecutionItem, error) {
	// we only start with go template + sprig
	// todo: add support to access definitions blob -> readFromFile, read file
	tmpl, err := template.New("execution").Funcs(sprig.FuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs)).Parse(blueprint.Info.Executors)
	if err != nil {
		return nil, err
	}

	// marshal and unmarshal resolved component descriptor
	components, err := serializeResolvedComponentDescriptor(cdutils.ConvertFromComponentDescriptorList(o.ResolvedComponentDescriptor).Components)
	if err != nil {
		return nil, fmt.Errorf("error during serializing of the resolved component descriptor: %w", err)
	}

	values := map[string]interface{}{
		"imports":    imports,
		"components": components,
	}

	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}

	executions := make([]lsv1alpha1.ExecutionItem, 0)
	if err := yaml.Unmarshal(data.Bytes(), &executions); err != nil {
		return nil, err
	}

	return executions, nil
}

func serializeResolvedComponentDescriptor(list map[string]cdutils.MappedComponentDescriptor) (interface{}, error) {
	data, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}
