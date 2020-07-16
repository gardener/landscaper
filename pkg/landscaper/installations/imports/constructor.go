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

package imports

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/utils"
)

// NewConstructor creates a new Import Contructor.
func NewConstructor(op *installations.Operation, parent *installations.Installation, siblings ...*installations.Installation) *Constructor {
	return &Constructor{
		Operation: op,
		validator: NewValidator(op, parent, siblings...),

		parent:   parent,
		siblings: siblings,
	}
}

// Construct loads all imported data from the datasources (either installations or the landscape config)
// and creates the imported configuration.
func (c *Constructor) Construct(ctx context.Context, inst *installations.Installation) (interface{}, error) {
	var (
		fldPath = field.NewPath(inst.Info.Name)
		values  = make(map[string]interface{})
	)
	for i, importMapping := range inst.Info.Spec.Imports {
		impPath := fldPath.Index(i)
		// check if the parent also imports my import
		newValues, err := c.constructForMapping(ctx, impPath, inst, importMapping)
		if err != nil {
			return nil, err
		}

		values = utils.MergeMaps(values, newValues)
	}

	return values, nil
}

func (c *Constructor) constructForMapping(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping lsv1alpha1.DefinitionImportMapping) (map[string]interface{}, error) {
	values, err := c.tryToConstructFromStaticData(ctx, fldPath, inst, mapping)
	if err == nil {
		return values, nil
	}
	if !installations.IsImportNotFoundError(err) {
		return nil, err
	}

	if !c.IsRoot() {
		values, err = c.tryToConstructFromParent(ctx, fldPath, inst, mapping)
		if err == nil {
			return values, nil
		}
		if !installations.IsImportNotFoundError(err) {
			return nil, err
		}
	}

	return c.tryToConstructFromSiblings(ctx, fldPath, inst, mapping)
}

func (c *Constructor) tryToConstructFromStaticData(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping lsv1alpha1.DefinitionImportMapping) (map[string]interface{}, error) {
	if err := c.validator.checkStaticDataForMapping(ctx, fldPath, inst, mapping); err != nil {
		return nil, err
	}

	data, err := c.GetStaticData(ctx)
	if err != nil {
		return nil, err
	}

	var val interface{}
	if err := jsonpath.GetValue(mapping.From, data, &val); err != nil {
		// can not happen as it is already checked in checkStaticDataForMapping
		return nil, installations.NewImportNotFoundErrorf(err, "%s: import in landscape config not found", fldPath.String())
	}

	values, err := jsonpath.Construct(mapping.To, val)
	if err != nil {
		return nil, err
	}

	tor, err := utils.TypedObjectReferenceFromObject(c.Inst.Info, kubernetes.LandscaperScheme)
	if err != nil {
		return nil, err
	}
	inst.ImportStatus().Update(lsv1alpha1.ImportState{
		From:             mapping.From,
		To:               mapping.To,
		SourceRef:        tor,
		ConfigGeneration: inst.Info.Generation,
	})

	return values, err
}

func (c *Constructor) tryToConstructFromParent(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping lsv1alpha1.DefinitionImportMapping) (map[string]interface{}, error) {
	if err := c.validator.checkIfParentHasImportForMapping(fldPath, inst, mapping); err != nil {
		return nil, err
	}

	values, err := c.constructValuesFromSecret(ctx, fldPath, c.parent.Info.Status.ImportReference.NamespacedName(), mapping.DefinitionFieldMapping)
	if err != nil {
		return nil, err
	}

	tor, err := utils.TypedObjectReferenceFromObject(c.parent.Info, kubernetes.LandscaperScheme)
	if err != nil {
		return nil, err
	}
	inst.ImportStatus().Update(lsv1alpha1.ImportState{
		From:             mapping.From,
		To:               mapping.To,
		SourceRef:        tor,
		ConfigGeneration: c.parent.Info.Status.ConfigGeneration,
	})
	return values, nil
}

func (c *Constructor) tryToConstructFromSiblings(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping lsv1alpha1.DefinitionImportMapping) (map[string]interface{}, error) {

	for _, sibling := range c.siblings {
		sPath := fldPath.Child(sibling.Info.Name)
		values, err := c.tryToConstructFromSibling(ctx, sPath, inst, mapping, sibling)
		if err == nil {
			return values, nil
		}
		if !installations.IsImportNotFoundError(err) {
			return nil, err
		}
	}

	return nil, installations.NewImportNotFoundError("no sibling installation found to satisfy import", nil)
}

func (c *Constructor) tryToConstructFromSibling(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping lsv1alpha1.DefinitionImportMapping, sibling *installations.Installation) (map[string]interface{}, error) {
	if err := c.validator.checkIfSiblingHasImportForMapping(fldPath, inst, mapping, sibling); err != nil {
		return nil, err
	}

	exportMapping, err := sibling.GetExportMappingTo(mapping.From)
	if err != nil {
		return nil, err
	}

	values, err := c.constructValuesFromSecret(ctx, fldPath, sibling.Info.Status.ExportReference.NamespacedName(), lsv1alpha1.DefinitionFieldMapping{From: exportMapping.From, To: mapping.To})
	if err != nil {
		return nil, err
	}

	tor, err := utils.TypedObjectReferenceFromObject(sibling.Info, kubernetes.LandscaperScheme)
	if err != nil {
		return nil, err
	}
	inst.ImportStatus().Update(lsv1alpha1.ImportState{
		From:             mapping.From,
		To:               mapping.To,
		SourceRef:        tor,
		ConfigGeneration: sibling.Info.Status.ConfigGeneration,
	})
	return values, nil
}

func (c *Constructor) constructValuesFromSecret(ctx context.Context, fldPath *field.Path, key types.NamespacedName, mapping lsv1alpha1.DefinitionFieldMapping) (map[string]interface{}, error) {
	do, err := c.GetDataObjectFromSecret(ctx, key)
	if err != nil {
		return nil, err
	}

	var val interface{}
	if err := do.GetData(mapping.From, &val); err != nil {
		// can not happen as it is already checked in checkStaticDataForMapping
		return nil, installations.NewImportNotFoundErrorf(err, "%s: import in config not found", fldPath.String())
	}

	return jsonpath.Construct(mapping.To, val)
}

func (c *Constructor) IsRoot() bool {
	return c.parent == nil
}
