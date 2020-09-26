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
	"errors"
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

// NewConstructor creates a new Import Constructor.
func NewConstructor(op *installations.Operation) *Constructor {
	return &Constructor{
		Operation: op,
		validator: NewValidator(op),

		parent:   op.Context().Parent,
		siblings: op.Context().Siblings,
	}
}

// Construct loads all imported data from the data sources (either installations or the landscape config)
// and creates the imported configuration.
func (c *Constructor) Construct(ctx context.Context, inst *installations.Installation) (map[string]interface{}, error) {
	var (
		fldPath = field.NewPath(inst.Info.Name)
		imports = make(map[string]interface{})
	)

	// read imports and construct internal templating imports
	importedDataObjects, err := c.GetImportedDataObjects(ctx)
	if err != nil {
		return nil, err
	}
	importedTargets, err := c.GetImportedTargets(ctx)
	if err != nil {
		return nil, err
	}

	templatedDataMappings, err := c.templateDataMappings(fldPath, importedDataObjects, importedTargets)
	if err != nil {
		return nil, err
	}

	// add additional imports and targets
	for _, def := range inst.Blueprint.Info.Imports {
		defPath := fldPath.Child(def.Name)
		if def.Schema != nil {
			if val, ok := templatedDataMappings[def.Name]; ok {
				imports[def.Name] = val
			} else if val, ok := importedDataObjects[def.Name]; ok {
				imports[def.Name] = val.Data
			}
			if _, ok := imports[def.Name]; !ok {
				return nil, installations.NewImportNotFoundErrorf(nil, "no import for %s exists", def.Name)
			}
			if err := c.JSONSchemaValidator().ValidateGoStruct(def.Schema, imports[def.Name]); err != nil {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported datatype does not have the expected schema", defPath.String())
			}
			continue
		}
		if len(def.TargetType) != 0 {
			if val, ok := templatedDataMappings[def.Name]; ok {
				imports[def.Name] = val
			} else if val, ok := importedTargets[def.Name]; ok {
				imports[def.Name], err = val.GetData()
				if err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported target cannot be parsed", defPath.String())
				}
			}
			data, ok := imports[def.Name]
			if !ok {
				return nil, installations.NewImportNotFoundErrorf(nil, "no import for %s exists", def.Name)
			}

			var targetType string
			if err := jsonpath.GetValue(".spec.type", data, &targetType); err != nil {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: exported target does not match the expected target template schema", defPath.String())
			}
			if def.TargetType != targetType {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, nil, "%s: exported target type is %s but expected %s", defPath.String(), targetType, def.TargetType)
			}
			continue
		}
		return nil, errors.New("whether a target nor a schema is defined")
	}

	return imports, nil
}

func (c *Constructor) templateDataMappings(fldPath *field.Path, importedDataObjects map[string]*dataobjects.DataObject, importedTargets map[string]*dataobjects.Target) (map[string]interface{}, error) {
	templateValues := map[string]interface{}{}
	for name, do := range importedDataObjects {
		templateValues[name] = do.Data
	}
	for name, target := range importedTargets {
		var err error
		templateValues[name], err = target.GetData()
		if err != nil {
			return nil, fmt.Errorf("unable to get target data for import %s", name)
		}
	}
	spiff, err := spiffing.New().WithFunctions(spiffing.NewFunctions()).WithValues(templateValues)
	if err != nil {
		return nil, fmt.Errorf("unable to init spiff templater: %w", err)
	}

	values := make(map[string]interface{})
	for key, dataMapping := range c.Inst.Info.Spec.ImportDataMappings {
		impPath := fldPath.Child(key)

		tmpl, err := spiffyaml.Unmarshal(key, dataMapping)
		if err != nil {
			return nil, fmt.Errorf("unable to parse import mapping template %s: %w", impPath.String(), err)
		}

		res, err := spiff.Cascade(tmpl, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to template import mapping template %s: %w", impPath.String(), err)
		}
		values[key] = res.Value()
	}
	return values, nil
}

//func (c *Constructor) constructForMapping(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) (*dataobjects.DataObject, error) {
//	do, err := c.tryToConstructFromStaticData(ctx, fldPath, inst, mapping)
//	if err == nil {
//		return do, nil
//	}
//	if !installations.IsImportNotFoundError(err) {
//		return nil, err
//	}
//
//	// get deploy item from current context
//	raw := &lsv1alpha1.DataObject{}
//	doName := lsv1alpha1helper.GenerateDataObjectName(c.Context().Name, mapping.From)
//	if err := c.Client().Get(ctx, kutil.ObjectKey(doName, inst.Info.Namespace), raw); err != nil {
//		return nil, err
//	}
//	do, err = dataobjects.NewFromDataObject(raw)
//	if err != nil {
//		return nil, err
//	}
//	// set new import metadata
//	do.SetSourceType(lsv1alpha1.ImportDataObjectSourceType)
//	do.SetKey(mapping.To)
//
//	if err := c.updateImportStateForDatatObject(ctx, inst, mapping, do); err != nil {
//		return nil, err
//	}
//
//	return do, nil
//}
//
//func (c *Constructor) tryToConstructFromStaticData(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) (*dataobjects.DataObject, error) {
//	if err := c.validator.checkStaticDataForMapping(ctx, fldPath, inst, mapping); err != nil {
//		return nil, err
//	}
//
//	data, err := c.GetStaticData(ctx)
//	if err != nil {
//		return nil, err
//	}
//
//	var val interface{}
//	if err := jsonpath.GetValue(mapping.From, data, &val); err != nil {
//		// should not happen as it is already checked in checkStaticDataForMapping
//		return nil, installations.NewImportNotFoundErrorf(err, "%s: import in landscape config not found", fldPath.String())
//	}
//
//	var encData bytes.Buffer
//	if err := gob.NewEncoder(&encData).Encode(val); err != nil {
//		return nil, err
//	}
//	h := sha1.New()
//	h.Write(encData.Bytes())
//
//	inst.ImportStatus().Update(lsv1alpha1.ImportState{
//		From: mapping.From,
//		To:   mapping.To,
//		SourceRef: &lsv1alpha1.ObjectReference{
//			Name:      inst.Info.Name,
//			Namespace: inst.Info.Namespace,
//		},
//		ConfigGeneration: fmt.Sprintf("%x", h.Sum(nil)),
//	})
//	do := dataobjects.New().SetSourceType(lsv1alpha1.ImportDataObjectSourceType).SetKey(mapping.To).SetData(val)
//	return do, err
//}
//
//func (c *Constructor) updateImportStateForDatatObject(ctx context.Context, inst *installations.Installation, mapping installations.ImportMapping, do *dataobjects.DataObject) error {
//	state := lsv1alpha1.ImportState{
//		From: mapping.From,
//		To:   mapping.To,
//	}
//	owner := kutil.GetOwner(do.Raw.ObjectMeta)
//	var ref *lsv1alpha1.ObjectReference
//	if owner != nil {
//		ref = &lsv1alpha1.ObjectReference{
//			Name:      owner.Name,
//			Namespace: inst.Info.Namespace,
//		}
//	}
//	state.SourceRef = ref
//
//	if owner != nil && owner.Kind == "Installation" {
//		inst := &lsv1alpha1.Installation{}
//		if err := c.Client().Get(ctx, ref.NamespacedName(), inst); err != nil {
//			return fmt.Errorf("unable to fetch source of data object for import %s: %w", mapping.Name, err)
//		}
//		state.ConfigGeneration = inst.Status.ConfigGeneration
//	}
//
//	inst.ImportStatus().Update(state)
//	return nil
//}

func (c *Constructor) IsRoot() bool {
	return c.parent == nil
}
