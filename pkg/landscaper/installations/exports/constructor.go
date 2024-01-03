// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package exports

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	genericresolver "github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver/generic"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"
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
func (c *Constructor) Construct(ctx context.Context) ([]*dataobjects.DataObject, []*dataobjects.TargetExtension, error) {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(c.Inst.GetInstallation()).String()})

	var (
		fldPath         = field.NewPath(fmt.Sprintf("(inst: %s)", c.Inst.GetInstallation().Name)).Child("internalExports")
		internalExports = map[string]interface{}{
			"deployitems": map[string]interface{}{},
			"dataobjects": map[string]interface{}{},
			"targets":     map[string]interface{}{},
		}
	)

	execDo, err := executions.New(c.Operation).GetExportedValues(ctx, c.Inst)
	if err != nil {
		return nil, nil, err
	}
	if execDo != nil {
		internalExports["deployitems"] = execDo.Data
	}

	dataObjectMap, err := c.aggregateDataObjectsInContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to aggregate data object: %w", err)
	}
	internalExports["dataobjects"] = dataObjectMap
	targetsMap, err := c.aggregateTargetsInContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to aggregate target: %w", err)
	}
	internalExports["targets"] = targetsMap

	stateHdlr := template.KubernetesStateHandler{
		KubeClient: c.Client(),
		Inst:       c.Inst.GetInstallation(),
	}
	targetResolver := genericresolver.New(c.Client())

	tmpl := template.New(
		gotemplate.New(stateHdlr, targetResolver),
		spiff.New(stateHdlr, targetResolver))
	exports, err := tmpl.TemplateExportExecutions(
		template.NewExportExecutionOptions(
			template.NewBlueprintExecutionOptions(
				c.Inst.GetInstallation(),
				c.Inst.GetBlueprint(),
				c.ComponentVersion,
				c.ResolvedComponentDescriptorList,
				c.Inst.GetImports()), internalExports))
	if err != nil {
		return nil, nil, err
	}

	// validate all exports
	for name := range exports {
		def, err := c.Inst.GetExportDefinition(name)
		if err != nil {
			// ignore additional exports
			logger.Info("key exported that is not defined by the blueprint", "name", name)
			delete(exports, name)
			continue
		}
		data := exports[name]

		switch def.Type {
		case lsv1alpha1.ExportTypeData:
			if def.Schema == nil {
				return nil, nil, fmt.Errorf("%s: schema for data export %q must not be empty", fldPath.String(), def.Name)
			}

			validator, err := c.JSONSchemaValidator(def.Schema.RawMessage)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: validator creation failed: %s", fldPath.String(), err.Error())
			}
			if err := validator.ValidateGoStruct(data); err != nil {
				return nil, nil, fmt.Errorf("%s: exported data does not satisfy the configured schema: %s", fldPath.String(), err.Error())
			}
		case lsv1alpha1.ExportTypeTarget:
			var targetType string
			if err := jsonpath.GetValue(".type", data, &targetType); err != nil {
				return nil, nil, fmt.Errorf("%s: exported target does not match the expected target template schema: %w", fldPath.String(), err)
			}
			if def.TargetType != targetType {
				return nil, nil, fmt.Errorf("%s: exported target type is %s but expected %s", fldPath.String(), targetType, def.TargetType)
			}
		default:
			return nil, nil, fmt.Errorf("%s: unknown export type '%s'", fldPath.String(), string(def.Type))
		}
	}

	// exportDataMappings
	if c.Inst.GetInstallation().Spec.ExportDataMappings != nil && len(c.Inst.GetInstallation().Spec.ExportDataMappings) > 0 {
		exportDataMappings, err := c.templateDataMappings(fldPath, exports)
		if err != nil {
			return nil, nil, err
		}
		// add exportDataMappings to available exports, potentially overwriting existing exports with that name
		for expName, expValue := range exportDataMappings {
			exports[expName] = expValue
		}
	}

	// Resolve export mapping for all internalExports
	dataObjects := make([]*dataobjects.DataObject, len(c.Inst.GetInstallation().Spec.Exports.Data))
	dataExportsPath := fldPath.Child("exports").Child("data")
	for i, dataExport := range c.Inst.GetInstallation().Spec.Exports.Data {
		dataExportPath := dataExportsPath.Child(dataExport.Name)
		data, ok := exports[dataExport.Name]
		if !ok {
			return nil, nil, fmt.Errorf("%s: data export is not defined", dataExportPath.String())
		}
		do := dataobjects.New().
			SetSourceType(lsv1alpha1.ExportDataObjectSourceType).
			SetKey(dataExport.DataRef).
			SetData(data)
		dataObjects[i] = do
	}

	targets := make([]*dataobjects.TargetExtension, len(c.Inst.GetInstallation().Spec.Exports.Targets))
	targetExportsPath := fldPath.Child("exports").Child("targets")
	for i, targetExport := range c.Inst.GetInstallation().Spec.Exports.Targets {
		targetExportPath := targetExportsPath.Child(targetExport.Name)
		data, ok := exports[targetExport.Name]
		if !ok {
			return nil, nil, fmt.Errorf("%s: target export is not defined", targetExportPath.String())
		}
		target, err := ConvertTargetTemplateToTargetExtension(data)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: unable to build target from template: %w", targetExportPath.String(), err)
		}
		target.SetSourceType(lsv1alpha1.ExportDataObjectSourceType).
			SetKey(targetExport.Target)
		targets[i] = target
	}

	return dataObjects, targets, nil
}

func (c *Constructor) aggregateDataObjectsInContext(ctx context.Context) (map[string]interface{}, error) {
	installationContext := lsv1alpha1helper.DataObjectSourceFromInstallation(c.Inst.GetInstallation())
	dataObjectList := &lsv1alpha1.DataObjectList{}
	if err := read_write_layer.ListDataObjects(ctx, c.Client(), dataObjectList, read_write_layer.R000070,
		client.InNamespace(c.Inst.GetInstallation().Namespace),
		client.MatchingLabels{lsv1alpha1.DataObjectContextLabel: installationContext}); err != nil {
		return nil, err
	}

	aggDataObjects := map[string]interface{}{}
	for _, do := range dataObjectList.Items {
		meta := dataobjects.GetMetadataFromObject(&do, do.Data.RawMessage)
		var data interface{}
		if err := yaml.Unmarshal(do.Data.RawMessage, &data); err != nil {
			return nil, fmt.Errorf("error while decoding data object %s: %w", do.Name, err)
		}
		aggDataObjects[meta.Key] = data
	}
	return aggDataObjects, nil
}

func (c *Constructor) aggregateTargetsInContext(ctx context.Context) (map[string]interface{}, error) {
	installationContext := lsv1alpha1helper.DataObjectSourceFromInstallation(c.Inst.GetInstallation())
	targetList := &lsv1alpha1.TargetList{}
	if err := read_write_layer.ListTargets(ctx, c.Client(), targetList, read_write_layer.R000071,
		client.InNamespace(c.Inst.GetInstallation().Namespace),
		client.MatchingLabels{lsv1alpha1.DataObjectContextLabel: installationContext}); err != nil {
		return nil, err
	}

	aggTargets := map[string]interface{}{}
	for _, target := range targetList.Items {
		meta := dataobjects.GetMetadataFromObject(&target, dataobjects.GetHashableContent(&target))
		raw, err := json.Marshal(target)
		if err != nil {
			return nil, fmt.Errorf("error while encoding target %s: %w", target.Name, err)
		}
		var data interface{}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, fmt.Errorf("error while decoding target %s: %w", target.Name, err)
		}
		aggTargets[meta.Key] = data
	}
	return aggTargets, nil
}

func ConvertTargetTemplateToTargetExtension(tmplData interface{}) (*dataobjects.TargetExtension, error) {
	data, err := json.Marshal(tmplData)
	if err != nil {
		return nil, err
	}
	targetTemplate := &lsv1alpha1.TargetTemplate{}
	if err := json.Unmarshal(data, targetTemplate); err != nil {
		return nil, err
	}
	target := &lsv1alpha1.Target{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      targetTemplate.Labels,
			Annotations: targetTemplate.Annotations,
		},
		Spec: lsv1alpha1.TargetSpec{
			Type:          targetTemplate.Type,
			Configuration: targetTemplate.Configuration,
		},
	}
	return dataobjects.NewTargetExtension(target, nil), nil
}

// templateDataMappings constructs the export data mappings
func (c *Constructor) templateDataMappings(fldPath *field.Path, exports map[string]interface{}) (map[string]interface{}, error) {

	// have all exports as top-level key and below an 'exports' key
	// this allows easy access, but forbids using exports which are named 'exports'
	exportValues := map[string]interface{}{}
	for k, v := range exports {
		exportValues[k] = v
	}
	exportValues["exports"] = exports
	spiff, err := spiffing.New().WithFunctions(spiffing.NewFunctions()).WithValues(exportValues)
	if err != nil {
		return nil, fmt.Errorf("unable to init spiff templater: %w", err)
	}

	values := make(map[string]interface{})
	for key, dataMapping := range c.Inst.GetInstallation().Spec.ExportDataMappings {
		expPath := fldPath.Child(key)

		tmpl, err := spiffyaml.Unmarshal(key, dataMapping.RawMessage)
		if err != nil {
			return nil, fmt.Errorf("unable to parse export mapping template %s: %w", expPath.String(), err)
		}

		res, err := spiff.Cascade(tmpl, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to template export mapping template %s: %w", expPath.String(), err)
		}

		dataBytes, err := spiffyaml.Marshal(res)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal templated export mapping %s: %w", expPath.String(), err)
		}
		var data interface{}
		if err := yaml.Unmarshal(dataBytes, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal templated export mapping %s: %w", expPath.String(), err)
		}
		values[key] = data
	}
	return values, nil
}
