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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
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

	mappings, err := c.Inst.GetExportMappings()
	if err != nil {
		return nil, err
	}

	// Resolve export mapping for all internalExports
	dataObjects := make([]*dataobjects.DataObject, len(mappings)) // map of mapping key to data
	for i, mapping := range mappings {

		data, ok := exports[mapping.From]
		if !ok {
			return nil, fmt.Errorf("%s: export %s is not defined", fldPath.String(), mapping.Key)
		}

		do := &dataobjects.DataObject{
			FieldValue: mapping.DefinitionFieldValue.DeepCopy(),
			Data:       data,
		}
		do.SetSourceType(lsv1alpha1.ExportDataObjectSourceType)
		do.SetKey(mapping.DefinitionExportMapping.To)

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
	do.SetSourceType(lsv1alpha1.ExportDataObjectSourceType)
	do.SetKey(mapping.DefinitionExportMapping.To)
	return do, nil
}

func (c *Constructor) constructFromSubInstallations(ctx context.Context, fldPath *field.Path, mapping installations.ExportMapping, subInsts []*installations.Installation) (*dataobjects.DataObject, error) {
	for _, subInst := range subInsts {
		subPath := fldPath.Child(fmt.Sprintf("(subinst: %s)", subInst.Info.Name))
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
	// todo: check if subinstallation exports this key, before reading it
	if _, err := subInst.GetExportMappingTo(mapping.From); err != nil {
		return nil, installations.NewErrorWrap(installations.ExportNotFound, field.NotFound(fldPath, "subinstallation does not export that key"))
	}

	// get specific export dataobject for the given exported key
	// subinst -> export -> key=mappingTo
	do, err := c.Operation.GetExportForKey(ctx, mapping.From)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, installations.NewExportNotFoundErrorf(err, "%s: could not fetch data object", fldPath.String())
		}
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
	do.SetSourceType(lsv1alpha1.ExportDataObjectSourceType)
	do.SetKey(mapping.DefinitionExportMapping.To)
	return do, nil
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
