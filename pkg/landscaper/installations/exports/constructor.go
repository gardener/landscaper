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

	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	"github.com/gardener/landscaper/pkg/utils"
)

// Constructor is a struct that contains all values
// that are needed to load and merge all exported data.
type Constructor struct {
	installations.Operation
}

// NewConstructor creates a new export constructor
func NewConstructor(op installations.Operation) *Constructor {
	return &Constructor{
		Operation: op,
	}
}

// Construct loads the exported data from the execution and the subinstallations.
func (c *Constructor) Construct(ctx context.Context, inst *installations.Installation) (map[string]interface{}, error) {
	var (
		fldPath   = field.NewPath(inst.Info.Name)
		allValues = make(map[string]interface{}, 0)
	)

	execDo, err := executions.New(c.Operation).GetExportedValues(ctx, inst)
	if err != nil {
		return nil, err
	}

	subInsts, err := c.getSubInstallations(ctx, inst)
	if err != nil {
		return nil, err
	}

	// Get export mapping for all exports
	for i, exportMapping := range inst.Info.Spec.Exports {
		exPath := fldPath.Index(i)
		values, err := c.constructFromExecution(inst, execDo, exportMapping)
		if err == nil {
			allValues = utils.MergeMaps(allValues, values)
			continue
		}
		if !installations.IsExportNotFoundError(err) {
			return nil, err
		}

		values, err = c.constructFromSubInstallations(ctx, exPath, exportMapping, subInsts)
		if err != nil {
			return nil, err
		}
		allValues = utils.MergeMaps(allValues, values)
	}

	return allValues, nil
}

func (c *Constructor) getSubInstallations(ctx context.Context, inst *installations.Installation) ([]*installations.Installation, error) {
	subInstsMap, err := subinstallations.New(c.Operation).GetSubInstallations(ctx, inst.Info)
	if err != nil {
		return nil, err
	}

	subInsts := make([]*installations.Installation, 0)
	for _, inst := range subInstsMap {
		inInst, err := installations.CreateInternalInstallation(c.Operation.Registry(), inst)
		if err != nil {
			return nil, err
		}
		subInsts = append(subInsts, inInst)
	}
	return subInsts, nil
}

func (c *Constructor) constructFromExecution(inst *installations.Installation, do *dataobject.DataObject, mapping lsv1alpha1.DefinitionExportMapping) (map[string]interface{}, error) {
	if do == nil {
		return nil, installations.NewExportNotFoundError("no execution dataobject given", nil)
	}

	var val interface{}
	if err := do.GetData(mapping.From, &val); err != nil {
		return nil, installations.NewExportNotFoundError("unable to find export from execution", err)
	}

	return jsonpath.Construct(mapping.To, val)
}

func (c *Constructor) constructFromSubInstallations(ctx context.Context, fldPath *field.Path, mapping lsv1alpha1.DefinitionExportMapping, subInsts []*installations.Installation) (map[string]interface{}, error) {
	for _, subInst := range subInsts {
		subPath := fldPath.Child(subInst.Info.Name)
		values, err := c.constructFromSubInstallation(ctx, subPath, mapping, subInst)
		if err == nil {
			return values, err
		}
		if !installations.IsExportNotFoundError(err) {
			return nil, err
		}
	}
	return nil, nil
}

func (c *Constructor) constructFromSubInstallation(ctx context.Context, fldPath *field.Path, mapping lsv1alpha1.DefinitionExportMapping, subInst *installations.Installation) (map[string]interface{}, error) {
	exportMapping, err := subInst.GetExportMappingTo(mapping.From)
	if err != nil {
		return nil, installations.NewExportNotFoundErrorf(err, "%s: unable to get export mapping form subinstllation", fldPath.String())
	}

	if subInst.Info.Status.ExportReference == nil {
		return nil, installations.NewExportNotFoundErrorf(err, "%s: subinstalation has no exported data", fldPath.String())
	}

	do, err := c.Operation.GetDataObjectFromSecret(ctx, subInst.Info.Status.ExportReference.NamespacedName())
	if err != nil {
		return nil, err
	}

	var val interface{}
	if err := do.GetData(exportMapping.From, &val); err != nil {
		return nil, err
	}

	return jsonpath.Construct(mapping.To, val)
}
