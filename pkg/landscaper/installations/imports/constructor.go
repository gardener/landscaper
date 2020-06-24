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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/landscapeconfig"
	"github.com/gardener/landscaper/pkg/utils"
)

func NewConstructor(op installations.Operation, landscapeConfig *landscapeconfig.LandscapeConfig, parent *installations.Installation, siblings ...*installations.Installation) *Constructor {
	return &Constructor{
		Operation: op,
		validator: NewValidator(op, landscapeConfig, parent, siblings...),

		lsConfig: landscapeConfig,
		parent:   parent,
		siblings: siblings,
	}
}

// Construct loads all imported data from the datasources (either installations or the landscape config)
// and creates the imported configuration.
func (c *Constructor) Construct(ctx context.Context, inst *installations.Installation) (interface{}, error) {
	var (
		fldPath = field.NewPath(inst.Info.Name)
		values  = make(map[string]interface{}, 0)
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
	if c.IsRoot() {
		values, err := c.tryToConstructFromLandscapeConfig(ctx, fldPath, inst, mapping)
		if err == nil {
			return values, nil
		}
		if !IsImportNotFoundError(err) {
			return nil, err
		}
	} else {
		values, err := c.tryToConstructFromParent(ctx, fldPath, inst, mapping)
		if err == nil {
			return values, nil
		}
		if !IsImportNotFoundError(err) {
			return nil, err
		}
	}

	return c.tryToConstructFromSiblings(ctx, fldPath, inst, mapping)
}

func (c *Constructor) tryToConstructFromLandscapeConfig(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping lsv1alpha1.DefinitionImportMapping) (map[string]interface{}, error) {
	if err := c.validator.checkIfLandscapeConfigForMapping(fldPath, inst, mapping); err != nil {
		return nil, err
	}

	var val interface{}
	if err := c.lsConfig.Data.GetData(mapping.From, &val); err != nil {
		// can not happen as it is already checked in checkIfLandscapeConfigForMapping
		return nil, NewImportNotFoundErrorf(err, "%s: import in landscape config not found", fldPath.String())
	}

	values, err := jsonpath.Construct(mapping.To, val)
	if err != nil {
		return nil, err
	}

	tor, err := utils.TypedObjectReferenceFromObject(c.lsConfig.Info, kubernetes.LandscaperScheme)
	if err != nil {
		return nil, err
	}
	inst.ImportStatus().Update(lsv1alpha1.ImportState{
		From:             mapping.From,
		To:               mapping.To,
		SourceRef:        tor,
		ConfigGeneration: c.lsConfig.Info.Status.ConfigGeneration,
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
		ConfigGeneration: c.lsConfig.Info.Status.ConfigGeneration,
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
		if !IsImportNotFoundError(err) {
			return nil, err
		}
	}

	return nil, NewImportNotFoundError("no sibling installation found to satisfy import", nil)
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
	secret := &corev1.Secret{}
	if err := c.Client().Get(ctx, key, secret); err != nil {
		return nil, err
	}

	do, err := dataobject.New(secret)
	if err != nil {
		return nil, err
	}

	var val interface{}
	if err := do.GetData(mapping.From, &val); err != nil {
		// can not happen as it is already checked in checkIfLandscapeConfigForMapping
		return nil, NewImportNotFoundErrorf(err, "%s: import in config not found", fldPath.String())
	}

	return jsonpath.Construct(mapping.To, val)
}

func (c *Constructor) IsRoot() bool {
	return c.parent == nil
}
