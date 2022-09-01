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
		siblings:  op.Context().Siblings,
	}
}

// Imports is a helper struct to pass around the loaded imports.
type Imports struct {
	DataObjects              map[string]*dataobjects.DataObject
	Targets                  map[string]*dataobjects.Target
	TargetLists              map[string]*dataobjects.TargetList
	ComponentDescriptors     map[string]*dataobjects.ComponentDescriptor
	ComponentDescriptorLists map[string]*dataobjects.ComponentDescriptorList
}

func (imps *Imports) All() []*dataobjects.Imported {
	res := make([]*dataobjects.Imported, 0, imps.Size())
	for impName, elem := range imps.DataObjects {
		res = append(res, dataobjects.NewImported(impName, elem))
	}
	for impName, elem := range imps.Targets {
		res = append(res, dataobjects.NewImported(impName, elem))
	}
	for impName, elem := range imps.TargetLists {
		res = append(res, dataobjects.NewImported(impName, elem))
	}
	for impName, elem := range imps.ComponentDescriptors {
		res = append(res, dataobjects.NewImported(impName, elem))
	}
	for impName, elem := range imps.ComponentDescriptorLists {
		res = append(res, dataobjects.NewImported(impName, elem))
	}

	return res
}

// Size returns the total amount of imports.
func (imps *Imports) Size() int {
	return len(imps.DataObjects) + len(imps.Targets) + len(imps.TargetLists) + len(imps.ComponentDescriptors) + len(imps.ComponentDescriptorLists)
}

// LoadImports loads all imports from the cluster (or wherever).
func (c *Constructor) LoadImports(ctx context.Context) (*Imports, error) {
	imps := &Imports{}
	var err error

	imps.DataObjects, err = c.GetImportedDataObjects(ctx) // returns a map mapping logical names to data objects
	if err != nil {
		return nil, err
	}
	imps.Targets, err = c.GetImportedTargets(ctx) // returns a map mapping logical names to targets
	if err != nil {
		return nil, err
	}
	imps.TargetLists, err = c.GetImportedTargetLists(ctx) // returns a map mapping logical names to target lists
	if err != nil {
		return nil, err
	}
	imps.ComponentDescriptors, err = c.GetImportedComponentDescriptors(ctx) // returns a map mapping logical names to component descriptors
	if err != nil {
		return nil, err
	}
	imps.ComponentDescriptorLists, err = c.GetImportedComponentDescriptorLists(ctx) // returns a map mapping logical names to lists of component descriptors
	if err != nil {
		return nil, err
	}
	return imps, nil
}

// Construct loads all imported data from the data sources (either installations or the landscape config)
// and creates the imported configuration.
// The imported data is added to installation resource.
func (c *Constructor) Construct(ctx context.Context, imps *Imports) error {
	inst := c.Inst
	fldPath := field.NewPath(inst.Info.Name)

	// if imports are not given, load them
	if imps == nil {
		var err error
		imps, err = c.LoadImports(ctx)
		if err != nil {
			return err
		}
	}

	templatedDataMappings, err := c.templateDataMappings(fldPath, imps.DataObjects, imps.Targets, imps.TargetLists, imps.ComponentDescriptors, imps.ComponentDescriptorLists) // returns a map mapping logical names to data content
	if err != nil {
		return err
	}

	// add additional imports and targets
	imports, err := c.constructImports(inst.Blueprint.Info.Imports, imps.DataObjects, imps.Targets, imps.TargetLists, imps.ComponentDescriptors, imps.ComponentDescriptorLists, templatedDataMappings, fldPath)
	if err != nil {
		return err
	}

	c.SetTargetImports(imps.Targets)
	c.SetTargetListImports(imps.TargetLists)

	inst.SetImports(imports)
	return nil
}

// constructImports is an auxiliary function that can be called in a recursive manner to traverse the tree of conditional imports
func (c *Constructor) constructImports(
	importList lsv1alpha1.ImportDefinitionList,
	importedDataObjects map[string]*dataobjects.DataObject,
	importedTargets map[string]*dataobjects.Target,
	importedTargetLists map[string]*dataobjects.TargetList,
	importedComponentDescriptors map[string]*dataobjects.ComponentDescriptor,
	importedComponentDescriptorLists map[string]*dataobjects.ComponentDescriptorList,
	templatedDataMappings map[string]interface{},
	fldPath *field.Path) (map[string]interface{}, error) {

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
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeData)
			}
			if def.Schema == nil {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, fmt.Errorf("schema is nil"), "%s: no schema defined", defPath.String())
			}
			validator, err := c.JSONSchemaValidator(def.Schema.RawMessage)
			if err != nil {
				return imports, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: validator creation failed", defPath.String())
			}
			if err := validator.ValidateGoStruct(imports[def.Name]); err != nil {
				return imports, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported datatype does not have the expected schema", defPath.String())
			}
			if len(def.ConditionalImports) > 0 {
				// recursively check conditional imports
				conditionalImports, err := c.constructImports(def.ConditionalImports, importedDataObjects, importedTargets, importedTargetLists, importedComponentDescriptors, importedComponentDescriptorLists, templatedDataMappings, defPath)
				if err != nil {
					return nil, err
				}
				for k, v := range conditionalImports {
					imports[k] = v
				}
			}
			continue
		case lsv1alpha1.ImportTypeTarget:
			if val, ok := importedTargets[def.Name]; ok {
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
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeTarget)
			}

			var targetType string
			if err := jsonpath.GetValue(".spec.type", data, &targetType); err != nil {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported target does not match the expected target template schema", defPath.String())
			}
			if def.TargetType != targetType {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, nil, "%s: imported target type is %s but expected %s", defPath.String(), targetType, def.TargetType)
			}
			continue
		case lsv1alpha1.ImportTypeTargetList:
			if val, ok := importedTargetLists[def.Name]; ok {
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
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeTargetList)
			}

			var targetType string
			listData, ok := data.([]interface{})
			if !ok {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, nil, "%s: targetlist import is not a list", defPath.String())
			}
			for i, elem := range listData {
				if err := jsonpath.GetValue(".spec.type", elem, &targetType); err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: element at position %d of the imported targetlist does not match the expected target template schema", defPath.String(), i)
				}
				if def.TargetType != targetType {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, nil, "%s: type of the element at position %d of the imported targetlist is %s but expected %s", defPath.String(), i, targetType, def.TargetType)
				}
			}
			continue
		case lsv1alpha1.ImportTypeComponentDescriptor:
			if val, ok := importedComponentDescriptors[def.Name]; ok {
				imports[def.Name], err = val.GetData()
				if err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported component descriptor cannot be parsed", defPath.String())
				}
			}
			_, ok := imports[def.Name]
			if !ok {
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeComponentDescriptor)
			}
			continue
		case lsv1alpha1.ImportTypeComponentDescriptorList:
			if val, ok := importedComponentDescriptorLists[def.Name]; ok {
				imports[def.Name], err = val.GetData()
				if err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported component descriptor list cannot be parsed", defPath.String())
				}
			}
			_, ok := imports[def.Name]
			if !ok {
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeComponentDescriptorList)
			}
			continue
		default:
			return nil, fmt.Errorf("%s: unknown import type '%s'", defPath.String(), string(def.Type))
		}
	}

	return imports, nil
}

func (c *Constructor) templateDataMappings(
	fldPath *field.Path,
	importedDataObjects map[string]*dataobjects.DataObject,
	importedTargets map[string]*dataobjects.Target,
	importedTargetLists map[string]*dataobjects.TargetList,
	importedComponentDescriptors map[string]*dataobjects.ComponentDescriptor,
	importedComponentDescriptorLists map[string]*dataobjects.ComponentDescriptorList) (map[string]interface{}, error) {

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
	for name, targetlist := range importedTargetLists {
		var err error
		templateValues[name], err = targetlist.GetData()
		if err != nil {
			return nil, fmt.Errorf("unable to get targetlist data for import %s", name)
		}
	}
	for name, cd := range importedComponentDescriptors {
		var err error
		templateValues[name], err = cd.GetData()
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
