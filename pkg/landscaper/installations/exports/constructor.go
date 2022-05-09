// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package exports

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
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
func (c *Constructor) Construct(ctx context.Context) ([]*dataobjects.DataObject, []*dataobjects.Target, error) {
	var (
		fldPath         = field.NewPath(fmt.Sprintf("(inst: %s)", c.Inst.Info.Name)).Child("internalExports")
		internalExports = map[string]interface{}{
			"deployitems": struct{}{},
			"dataobjects": struct{}{},
			"targets":     struct{}{},
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
		Inst:       c.Inst.Info,
	}
	exports, err := template.New(gotemplate.New(c.BlobResolver, stateHdlr), spiff.New(stateHdlr)).
		TemplateExportExecutions(template.NewExportExecutionOptions(template.NewBlueprintExecutionOptions(
			c.Inst.GetInfo(), c.Inst.Blueprint, c.ComponentDescriptor, c.ResolvedComponentDescriptorList, c.Inst.GetImports()),
			internalExports))
	if err != nil {
		return nil, nil, err
	}

	// validate all exports
	for name := range exports {
		def, err := c.Inst.GetExportDefinition(name)
		if err != nil {
			// ignore additional exports
			c.Log().V(5).Info("key exported that is not defined by the blueprint", "name", name)
			delete(exports, name)
			continue
		}
		data := exports[name]

		switch def.Type {
		case lsv1alpha1.ExportTypeData:
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

	// Resolve export mapping for all internalExports
	dataObjects := make([]*dataobjects.DataObject, len(c.Inst.Info.Spec.Exports.Data))
	dataExportsPath := fldPath.Child("exports").Child("data")
	for i, dataExport := range c.Inst.Info.Spec.Exports.Data {
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

	targets := make([]*dataobjects.Target, len(c.Inst.Info.Spec.Exports.Targets))
	targetExportsPath := fldPath.Child("exports").Child("targets")
	for i, targetExport := range c.Inst.Info.Spec.Exports.Targets {
		targetExportPath := targetExportsPath.Child(targetExport.Name)
		data, ok := exports[targetExport.Name]
		if !ok {
			return nil, nil, fmt.Errorf("%s: target export is not defined", targetExportPath.String())
		}
		target, err := ConvertTargetTemplateToTarget(data)
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
	installationContext := lsv1alpha1helper.DataObjectSourceFromInstallation(c.Inst.Info)
	dataObjectList := &lsv1alpha1.DataObjectList{}
	if err := c.Client().List(ctx, dataObjectList, client.InNamespace(c.Inst.Info.Namespace), client.MatchingLabels{lsv1alpha1.DataObjectContextLabel: installationContext}); err != nil {
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
	installationContext := lsv1alpha1helper.DataObjectSourceFromInstallation(c.Inst.Info)
	targetList := &lsv1alpha1.TargetList{}
	if err := c.Client().List(ctx, targetList, client.InNamespace(c.Inst.Info.Namespace), client.MatchingLabels{lsv1alpha1.DataObjectContextLabel: installationContext}); err != nil {
		return nil, err
	}

	aggTargets := map[string]interface{}{}
	for _, target := range targetList.Items {
		meta := dataobjects.GetMetadataFromObject(&target, target.Spec.Configuration.RawMessage)
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

func ConvertTargetTemplateToTarget(tmplData interface{}) (*dataobjects.Target, error) {
	data, err := json.Marshal(tmplData)
	if err != nil {
		return nil, err
	}
	targetTemplate := &lsv1alpha1.TargetTemplate{}
	if err := json.Unmarshal(data, targetTemplate); err != nil {
		return nil, err
	}
	raw := &lsv1alpha1.Target{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      targetTemplate.Labels,
			Annotations: targetTemplate.Annotations,
		},
		Spec: lsv1alpha1.TargetSpec{
			Type:          targetTemplate.Type,
			Configuration: targetTemplate.Configuration,
		},
	}
	return dataobjects.NewFromTarget(raw)
}
