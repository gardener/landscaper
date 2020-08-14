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
	*installations.Operation
}

// NewConstructor creates a new export constructor
func NewConstructor(op *installations.Operation) *Constructor {
	return &Constructor{
		Operation: op,
	}
}

// Construct loads the exported data from the execution and the subinstallations.
func (c *Constructor) Construct(ctx context.Context, inst *installations.Installation) (map[string]interface{}, error) {
	var (
		fldPath   = field.NewPath(fmt.Sprintf("(inst: %s)", inst.Info.Name)).Child("exports")
		allValues = make(map[string]interface{})
	)

	execDo, err := executions.New(c.Operation).GetExportedValues(ctx, inst)
	if err != nil {
		return nil, err
	}

	subInsts, err := c.getSubInstallations(ctx, inst)
	if err != nil {
		return nil, err
	}

	mappings, err := inst.GetExportMappings()
	if err != nil {
		return nil, err
	}

	// Resolve export mapping for all exports
	for _, mapping := range mappings {
		exPath := fldPath.Child(mapping.Key)
		values, err := c.constructFromExecution(exPath, execDo, mapping)
		if err == nil {
			allValues = utils.MergeMaps(allValues, values)
			continue
		}
		if !installations.IsExportNotFoundError(err) {
			return nil, err
		}

		values, err = c.constructFromSubInstallations(ctx, exPath, mapping, subInsts)
		if err != nil {
			return nil, err
		}
		if values == nil {
			return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(exPath, "no data found"))
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
		inInst, err := installations.CreateInternalInstallation(ctx, c.Operation.Registry(), inst)
		if err != nil {
			return nil, err
		}
		subInsts = append(subInsts, inInst)
	}
	return subInsts, nil
}

func (c *Constructor) constructFromExecution(fldPath *field.Path, do *dataobject.DataObject, mapping installations.ExportMapping) (map[string]interface{}, error) {
	if do == nil {
		return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(fldPath, "no execution dataobject given"))
	}

	var val interface{}
	if err := do.GetData(mapping.From, &val); err != nil {
		return nil, installations.NewExportNotFoundError("unable to find export from execution", field.InternalError(fldPath, err))
	}

	data, err := jsonpath.Construct(mapping.To, val)
	if err != nil {
		return nil, field.InternalError(fldPath, err)
	}
	return data, nil
}

func (c *Constructor) constructFromSubInstallations(ctx context.Context, fldPath *field.Path, mapping installations.ExportMapping, subInsts []*installations.Installation) (map[string]interface{}, error) {
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

func (c *Constructor) constructFromSubInstallation(ctx context.Context, fldPath *field.Path, mapping installations.ExportMapping, subInst *installations.Installation) (map[string]interface{}, error) {
	if subInst.Info.Status.ExportReference == nil {
		return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(fldPath, "subinstallation has no exported data"))
	}

	// todo: check if subinstalaltion exports this key, before reading it
	_, err := subInst.GetExportMappingTo(mapping.From)
	if err != nil {
		return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(fldPath, "subinstallation does not export that key"))
	}

	do, err := c.Operation.GetDataObjectFromSecret(ctx, subInst.Info.Status.ExportReference.NamespacedName())
	if err != nil {
		return nil, field.InternalError(fldPath, err)
	}

	var val interface{}
	if err := do.GetData(mapping.From, &val); err != nil {
		return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(fldPath, err))
	}

	data, err := jsonpath.Construct(mapping.To, val)
	if err != nil {
		return nil, field.InternalError(fldPath, err)
	}
	return data, nil
}
