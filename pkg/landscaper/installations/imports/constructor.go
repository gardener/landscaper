// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"context"
	"errors"
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

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

		dataBytes, err := spiffyaml.Marshal(res)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal templated import mapping %s: %w", impPath.String(), err)
		}
		var data interface{}
		if err := yaml.Unmarshal(dataBytes, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal templated import mapping %s: %w", impPath.String(), err)
		}
		values[key] = data
	}
	return values, nil
}

func (c *Constructor) IsRoot() bool {
	return c.parent == nil
}
