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

package exports

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
)

// Constructor is a struct that contains all values
// that are needed to load and merge all exported data.
type Constructor struct {
	*installations.Operation
}

// NewConstructor creates a new export constructor
func NewConstructor(op *installations.Operation) *Constructor {
	return &Constructor{
		Operation: op,
	}
}

// Construct loads the exported data from the execution and the subinstallations.
func (c *Constructor) Construct(ctx context.Context) ([]*dataobjects.DataObject, error) {
	var (
		fldPath         = field.NewPath(fmt.Sprintf("(inst: %s)", c.Inst.Info.Name)).Child("internalExports")
		bpPath          = fldPath.Child(fmt.Sprintf("(Bueprint %s)", c.Inst.Blueprint.Info.Name))
		internalExports = map[string]interface{}{
			"di": struct{}{},
			"do": struct{}{},
		}
	)

	execDo, err := executions.New(c.Operation).GetExportedValues(ctx, c.Inst)
	if err != nil {
		return nil, err
	}
	if execDo != nil {
		internalExports["di"] = execDo.Data
	}

	dataObjectMap, err := c.aggregateDataObjectsInContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to aggregate data object: %w", err)
	}
	internalExports["do"] = dataObjectMap

	templater := template.New(c.Operation)
	exports, err := templater.TemplateExportExecutions(c.Inst.Blueprint, internalExports)
	if err != nil {
		return nil, err
	}

	// validate all exports
	for name, data := range exports {
		def, err := c.Inst.GetExportDefinition(name)
		if err != nil {
			// ignore additional exports
			c.Log().V(5).Info("key exported that is not defined by the blueprint", "name", name)
			continue
		}

		if len(def.Schema) != 0 {
			if err := c.JSONSchemaValidator().ValidateGoStruct(def.Schema, data); err != nil {
				return nil, fmt.Errorf("%s: exported data does not satisfy the configured schema: %s", fldPath.String(), err.Error())
			}
		} else if len(def.TargetType) != 0 {
			return nil, fmt.Errorf("target type validation not implemented yet; check %s", bpPath.Child("exports").Child(name).String())
		}
	}

	// Resolve export mapping for all internalExports
	// todo: add target support
	dataObjects := make([]*dataobjects.DataObject, len(c.Inst.Info.Spec.Exports.Data))
	dataExportsPath := fldPath.Child("exports")
	for i, dataExport := range c.Inst.Info.Spec.Exports.Data {
		dataExportPath := dataExportsPath.Child(dataExport.Name)
		data, ok := exports[dataExport.Name]
		if !ok {
			return nil, fmt.Errorf("%s: export is not defined", dataExportPath.String())
		}
		do := dataobjects.New().
			SetSourceType(lsv1alpha1.ExportDataObjectSourceType).
			SetKey(dataExport.DataRef).
			SetData(data)
		dataObjects[i] = do
	}

	return dataObjects, nil
}

func (c *Constructor) getSubInstallations(ctx context.Context, inst *installations.Installation) ([]*installations.Installation, error) {
	subInstsMap, err := subinstallations.New(c.Operation).GetSubInstallations(ctx, inst.Info)
	if err != nil {
		return nil, err
	}

	subInsts := make([]*installations.Installation, 0)
	for _, inst := range subInstsMap {
		inInst, err := installations.CreateInternalInstallation(ctx, c.Operation, inst)
		if err != nil {
			return nil, err
		}
		subInsts = append(subInsts, inInst)
	}
	return subInsts, nil
}

func (c *Constructor) aggregateDataObjectsInContext(ctx context.Context) (map[string]interface{}, error) {
	installationContext := lsv1alpha1helper.DataObjectSourceFromInstallation(c.Inst.Info)
	dataObjectList := &lsv1alpha1.DataObjectList{}
	if err := c.Client().List(ctx, dataObjectList, client.InNamespace(c.Inst.Info.Namespace), client.MatchingLabels{lsv1alpha1.DataObjectContextLabel: installationContext}); err != nil {
		return nil, err
	}

	aggDataObjects := map[string]interface{}{}
	for _, do := range dataObjectList.Items {
		meta := dataobjects.GetMetadataFromObject(&do)
		var data interface{}
		if err := yaml.Unmarshal(do.Data, &data); err != nil {
			return nil, fmt.Errorf("error while decoding data object %s: %w", do.Name, err)
		}
		aggDataObjects[meta.Key] = data
	}
	return aggDataObjects, nil
}
