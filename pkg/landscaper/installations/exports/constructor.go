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

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
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
func (c *Constructor) Construct(ctx context.Context, inst *installations.Installation) ([]*dataobjects.DataObject, error) {
	var (
		fldPath = field.NewPath(fmt.Sprintf("(inst: %s)", inst.Info.Name)).Child("exports")
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
	dataObjects := make([]*dataobjects.DataObject, len(mappings)) // map of mapping key to data
	for i, mapping := range mappings {
		exPath := fldPath.Child(mapping.Key)

		if execDo != nil {
			do, err := c.constructFromExecution(exPath, execDo.Data, mapping)
			if err == nil {
				dataObjects[i] = do
				continue
			}
			if !installations.IsExportNotFoundError(err) {
				return nil, err
			}
		}

		do, err := c.constructFromSubInstallations(ctx, exPath, mapping, subInsts)
		if err != nil {
			return nil, err
		}
		if do == nil {
			return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(exPath, "no data found"))
		}
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

func (c *Constructor) constructFromExecution(fldPath *field.Path, input interface{}, mapping installations.ExportMapping) (*dataobjects.DataObject, error) {
	if input == nil {
		return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(fldPath, "no execution dataobject given"))
	}

	var data interface{}
	if err := jsonpath.GetValue(mapping.From, input, &data); err != nil {
		return nil, installations.NewExportNotFoundError("unable to find export from execution", field.InternalError(fldPath, err))
	}

	dt, ok := c.GetDataType(mapping.Type)
	if !ok {
		return nil, fmt.Errorf("%s: cannot find DataType %s", fldPath.String(), mapping.Type)
	}

	if err := datatype.Validate(*dt, data); err != nil {
		return nil, errors.Wrapf(err, "%s: unable to validate data against %s", fldPath.String(), mapping.Type)
	}

	do := &dataobjects.DataObject{
		FieldValue: &mapping.DefinitionFieldValue,
		Data:       data,
	}
	do.SetContext(lsv1alpha1.ExportDataObjectContext)
	do.SetKey(mapping.DefinitionExportMapping.To)
	return do, nil
}

func (c *Constructor) constructFromSubInstallations(ctx context.Context, fldPath *field.Path, mapping installations.ExportMapping, subInsts []*installations.Installation) (*dataobjects.DataObject, error) {
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

func (c *Constructor) constructFromSubInstallation(ctx context.Context, fldPath *field.Path, mapping installations.ExportMapping, subInst *installations.Installation) (*dataobjects.DataObject, error) {
	if subInst.Info.Status.ExportReference == nil {
		return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(fldPath, "subinstallation has no exported data"))
	}

	// todo: check if subinstallation exports this key, before reading it
	_, err := subInst.GetExportMappingTo(mapping.From)
	if err != nil {
		return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(fldPath, "subinstallation does not export that key"))
	}

	// get specific export dataobject for the given exported key
	// subinst -> export -> key=mappingTo
	do, err := c.Operation.GetExportForKey(ctx, subInst.Info, mapping.From)
	if err != nil {
		return nil, field.InternalError(fldPath, err)
	}

	dt, ok := c.GetDataType(mapping.Type)
	if !ok {
		return nil, fmt.Errorf("%s: cannot find DataType %s", fldPath.String(), mapping.Type)
	}

	if err := datatype.Validate(*dt, do.Data); err != nil {
		return nil, errors.Wrapf(err, "%s: unable to validate data against %s", fldPath.String(), mapping.Type)
	}

	do = &dataobjects.DataObject{
		FieldValue: &mapping.DefinitionFieldValue,
		Data:       do.Data,
	}
	do.SetContext(lsv1alpha1.ExportDataObjectContext)
	do.SetKey(mapping.DefinitionExportMapping.To)
	return do, nil
}
