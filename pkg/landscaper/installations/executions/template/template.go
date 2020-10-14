// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	"github.com/gardener/landscaper/pkg/utils"
)

// Templater implements all available template executors.
// todo: implement state handling
type Templater struct {
	lsoperation.Interface
	impl map[lsv1alpha1.TemplateType]templateExecution
}

// New creates a new instance of a templator.
func New(op lsoperation.Interface) *Templater {
	return &Templater{
		Interface: op,
		impl: map[lsv1alpha1.TemplateType]templateExecution{
			lsv1alpha1.GOTemplateType:    &GoTemplateExecution{},
			lsv1alpha1.SpiffTemplateType: &SpiffTemplate{},
		},
	}
}

// templateExecution describes a implementation for a template execution
type templateExecution interface {
	// TemplateDeployExecutions templates a deploy executor and return a list of deployitem templates.
	TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, components, imports interface{}) ([]byte, error)
	// TemplateExportExecutions templates a export executor.
	// It return the exported data as key value map where the key is the name of the export.
	TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) ([]byte, error)
}

// TemplateDeployExecutions templates all deploy executions and returns a aggregated list of all templated deploy item templates.
func (o *Templater) TemplateDeployExecutions(blueprint *blueprints.Blueprint, cd *cdutils.ResolvedComponentDescriptor, imports interface{}) ([]lsv1alpha1.DeployItemTemplate, error) {

	// marshal and unmarshal resolved component descriptor
	components, err := serializeResolvedComponentDescriptor(cd)
	if err != nil {
		return nil, fmt.Errorf("error during serializing of the resolved component descriptor: %w", err)
	}

	executionItems := make([]lsv1alpha1.DeployItemTemplate, 0)
	for _, tmplExec := range blueprint.Info.DeployExecutions {
		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		executionItemsBytes, err := impl.TemplateDeployExecutions(tmplExec, blueprint, components, imports)
		if err != nil {
			return nil, err
		}

		var items []lsv1alpha1.DeployItemTemplate
		if err := yaml.Unmarshal(executionItemsBytes, &items); err != nil {
			return nil, fmt.Errorf("error while decoding templated execution: %w", err)
		}
		executionItems = append(executionItems, items...)
	}

	return executionItems, nil
}

// TemplateDeployExecutions templates all deploy executions and returns a aggregated list of all templated deploy item templates.
func (o *Templater) TemplateExportExecutions(blueprint *blueprints.Blueprint, exports interface{}) (map[string]interface{}, error) {
	exportData := make(map[string]interface{})
	for _, tmplExec := range blueprint.Info.ExportExecutions {

		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		exportDataJSONBytes, err := impl.TemplateExportExecutions(tmplExec, blueprint, exports)
		if err != nil {
			return nil, err
		}
		var exportDataJSON map[string]interface{}
		if err := yaml.Unmarshal(exportDataJSONBytes, &exportDataJSON); err != nil {
			return nil, err
		}
		exportData = utils.MergeMaps(exportData, exportDataJSON)
	}

	return exportData, nil
}

func serializeResolvedComponentDescriptor(cd *cdutils.ResolvedComponentDescriptor) (interface{}, error) {
	data, err := json.Marshal(cd)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}
