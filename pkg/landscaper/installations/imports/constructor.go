// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"context"
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
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
// The imported data is added to installation resource.
func (c *Constructor) Construct(ctx context.Context, inst *installations.Installation) error {
	var (
		fldPath = field.NewPath(inst.Info.Name)
	)

	// read imports and construct internal templating imports
	importedDataObjects, err := c.GetImportedDataObjects(ctx) // returns a map mapping logical names to data objects
	if err != nil {
		return err
	}
	importedTargets, err := c.GetImportedTargets(ctx) // returns a map mapping logical names to targets
	if err != nil {
		return err
	}

	templatedDataMappings, err := c.templateDataMappings(fldPath, importedDataObjects, importedTargets) // returns a map mapping logical names to data content
	if err != nil {
		return err
	}

	// add additional imports and targets
	imports, err := c.constructImports(inst.Blueprint.Info.Imports, importedDataObjects, importedTargets, templatedDataMappings, fldPath)
	if err != nil {
		return err
	}

	inst.SetImports(imports)
	return nil
}

// constructImports is an auxiliary function that can be called in a recursive manner to traverse the tree of conditional imports
func (c *Constructor) constructImports(importList lsv1alpha1.ImportDefinitionList, importedDataObjects map[string]*dataobjects.DataObject, importedTargets map[string]*dataobjects.Target, templatedDataMappings map[string]interface{}, fldPath *field.Path) (map[string]interface{}, error) {
	imports := map[string]interface{}{}
	for _, def := range importList {
		var err error
		defPath := fldPath.Child(def.Name)
		switch def.Type {
		case lsv1alpha1.ImportTypeData:
			if val, ok := templatedDataMappings[def.Name]; ok {
				imports[def.Name] = val
			} else if val, ok := importedDataObjects[def.Name]; ok {
				imports[def.Name] = val.Data
			}
			if _, ok := imports[def.Name]; !ok {
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
				return nil, installations.NewImportNotFoundErrorf(nil, "no import for %s exists", def.Name)
			}
			if err := c.JSONSchemaValidator().ValidateGoStruct(def.Schema.RawMessage, imports[def.Name]); err != nil {
				return imports, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported datatype does not have the expected schema", defPath.String())
			}
			if len(def.ConditionalImports) > 0 {
				// recursively check conditional imports
				conditionalImports, err := c.constructImports(def.ConditionalImports, importedDataObjects, importedTargets, templatedDataMappings, defPath)
				if err != nil {
					return nil, err
				}
				for k, v := range conditionalImports {
					imports[k] = v
				}
			}
			continue
		case lsv1alpha1.ImportTypeTarget:
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
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
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
		default:
			return nil, fmt.Errorf("%s: unknown import type '%s'", defPath.String(), string(def.Type))
		}
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

		tmpl, err := spiffyaml.Unmarshal(key, dataMapping.RawMessage)
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
