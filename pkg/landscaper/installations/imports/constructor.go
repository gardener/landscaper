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
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/gob"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/utils"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// NewConstructor creates a new Import Constructor.
func NewConstructor(op *installations.Operation, parent *installations.Installation, siblings ...*installations.Installation) *Constructor {
	return &Constructor{
		Operation: op,
		validator: NewValidator(op, parent, siblings...),

		parent:   parent,
		siblings: siblings,
	}
}

// Construct loads all imported data from the data sources (either installations or the landscape config)
// and creates the imported configuration.
func (c *Constructor) Construct(ctx context.Context, inst *installations.Installation) ([]*dataobjects.DataObject, interface{}, error) {
	var (
		fldPath     = field.NewPath(inst.Info.Name)
		values      = make(map[string]interface{})
		mappings    = inst.GetImportMappings()
		dataObjects = make([]*dataobjects.DataObject, len(mappings))
	)

	for i, importMapping := range mappings {
		impPath := fldPath.Index(i)
		do, err := c.constructForMapping(ctx, impPath, inst, importMapping)
		if err != nil {
			return nil, nil, err
		}

		dataObjects = append(dataObjects, do)

		value, err := jsonpath.Construct(importMapping.To, do.Data)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to construct object with path %s for import %s: %w", importMapping.To, importMapping.Key, err)
		}
		values = utils.MergeMaps(values, value)
	}

	return dataObjects, values, nil
}

func (c *Constructor) constructForMapping(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) (*dataobjects.DataObject, error) {
	do, err := c.tryToConstructFromStaticData(ctx, fldPath, inst, mapping)
	if err == nil {
		return do, nil
	}
	if !installations.IsImportNotFoundError(err) {
		return nil, err
	}

	if !c.IsRoot() {
		do, err = c.tryToConstructFromParent(ctx, fldPath, inst, mapping)
		if err == nil {
			return do, nil
		}
		if !installations.IsImportNotFoundError(err) {
			return nil, err
		}
	}

	return c.tryToConstructFromSiblings(ctx, fldPath, inst, mapping)
}

func (c *Constructor) tryToConstructFromStaticData(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) (*dataobjects.DataObject, error) {
	if err := c.validator.checkStaticDataForMapping(ctx, fldPath, inst, mapping); err != nil {
		return nil, err
	}

	data, err := c.GetStaticData(ctx)
	if err != nil {
		return nil, err
	}

	var val interface{}
	if err := jsonpath.GetValue(mapping.From, data, &val); err != nil {
		// should not happen as it is already checked in checkStaticDataForMapping
		return nil, installations.NewImportNotFoundErrorf(err, "%s: import in landscape config not found", fldPath.String())
	}

	var encData bytes.Buffer
	if err := gob.NewEncoder(&encData).Encode(val); err != nil {
		return nil, err
	}
	h := sha1.New()
	h.Write(encData.Bytes())

	inst.ImportStatus().Update(lsv1alpha1.ImportState{
		From: mapping.From,
		To:   mapping.To,
		SourceRef: &lsv1alpha1.ObjectReference{
			Name:      inst.Info.Name,
			Namespace: inst.Info.Namespace,
		},
		ConfigGeneration: fmt.Sprintf("%x", h.Sum(nil)),
	})
	do := dataobjects.New().SetContext(lsv1alpha1.ImportDataObjectContext).SetKey(mapping.To).SetData(val)
	return do, err
}

func (c *Constructor) tryToConstructFromParent(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) (*dataobjects.DataObject, error) {
	if err := c.validator.checkIfParentHasImportForMapping(fldPath, mapping); err != nil {
		return nil, err
	}

	raw := &lsv1alpha1.DataObject{}
	doName := lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1.ImportDataObjectContext, lsv1alpha1helper.DataObjectSourceFromInstallation(c.parent.Info), mapping.From)
	if err := c.Client().Get(ctx, kutil.ObjectKey(doName, c.parent.Info.Namespace), raw); err != nil {
		return nil, err
	}

	do, err := dataobjects.NewFromDataObject(raw)
	if err != nil {
		return nil, err
	}
	do.SetContext(lsv1alpha1.ImportDataObjectContext)
	do.SetKey(mapping.To)

	inst.ImportStatus().Update(lsv1alpha1.ImportState{
		From: mapping.From,
		To:   mapping.To,
		SourceRef: &lsv1alpha1.ObjectReference{
			Name:      c.parent.Info.Name,
			Namespace: c.parent.Info.Namespace,
		},
		ConfigGeneration: c.parent.Info.Status.ConfigGeneration,
	})
	return do, nil
}

func (c *Constructor) tryToConstructFromSiblings(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) (*dataobjects.DataObject, error) {
	for _, sibling := range c.siblings {
		sPath := fldPath.Child(sibling.Info.Name)
		do, err := c.tryToConstructFromSibling(ctx, sPath, inst, mapping, sibling)
		if err == nil {
			return do, nil
		}
		if !installations.IsImportNotFoundError(err) {
			return nil, err
		}
	}

	return nil, installations.NewImportNotFoundError("no sibling installation found to satisfy import", nil)
}

func (c *Constructor) tryToConstructFromSibling(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping, sibling *installations.Installation) (*dataobjects.DataObject, error) {
	if err := c.validator.checkIfSiblingHasImportForMapping(ctx, fldPath, inst, mapping, sibling); err != nil {
		return nil, err
	}

	raw := &lsv1alpha1.DataObject{}
	doName := lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1.ExportDataObjectContext, lsv1alpha1helper.DataObjectSourceFromInstallation(sibling.Info), mapping.From)
	if err := c.Client().Get(ctx, kutil.ObjectKey(doName, c.parent.Info.Namespace), raw); err != nil {
		return nil, err
	}

	do, err := dataobjects.NewFromDataObject(raw)
	if err != nil {
		return nil, err
	}
	do.SetContext(lsv1alpha1.ImportDataObjectContext)
	do.SetKey(mapping.To)

	inst.ImportStatus().Update(lsv1alpha1.ImportState{
		From: mapping.From,
		To:   mapping.To,
		SourceRef: &lsv1alpha1.ObjectReference{
			Name:      sibling.Info.Name,
			Namespace: sibling.Info.Namespace,
		},
		ConfigGeneration: sibling.Info.Status.ConfigGeneration,
	})
	return do, nil
}

func (c *Constructor) IsRoot() bool {
	return c.parent == nil
}
