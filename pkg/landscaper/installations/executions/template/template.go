// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"encoding/json"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/validation"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/utils"
)

// Templater implements all available template executors.
type Templater struct {
	stateHdlr GenericStateHandler
	impl      map[lsv1alpha1.TemplateType]templateExecution
}

// New creates a new instance of a templater.
func New(blobResolver ctf.BlobResolver, state GenericStateHandler) *Templater {
	return &Templater{
		stateHdlr: state,
		impl: map[lsv1alpha1.TemplateType]templateExecution{
			lsv1alpha1.GOTemplateType: &GoTemplateExecution{
				blobResolver: blobResolver,
				state:        state,
			},
			lsv1alpha1.SpiffTemplateType: &SpiffTemplate{state: state},
		},
	}
}

// templateExecution describes a implementation for a template execution
type templateExecution interface {
	// TemplateDeployExecutions templates a deploy executor and return a list of deployitem templates.
	TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) ([]byte, error)
	// TemplateExportExecutions templates a export executor.
	// It return the exported data as key value map where the key is the name of the export.
	TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) ([]byte, error)
}

// DeployExecutorOutput describes the output of deploy executor.
type DeployExecutorOutput struct {
	DeployItems []lsv1alpha1.DeployItemTemplate `json:"deployItems"`
}

// DeployExecutionOptions describes the options for templating the deploy executions.
type DeployExecutionOptions struct {
	Imports interface{}
	// +optional
	Installation         *lsv1alpha1.Installation
	Blueprint            *blueprints.Blueprint
	ComponentDescriptor  *cdv2.ComponentDescriptor
	ComponentDescriptors *cdv2.ComponentDescriptorList
}

// TemplateDeployExecutions templates all deploy executions and returns a aggregated list of all templated deploy item templates.
func (o *Templater) TemplateDeployExecutions(opts DeployExecutionOptions) ([]lsv1alpha1.DeployItemTemplate, error) {

	// marshal and unmarshal resolved component descriptor
	component, err := serializeComponentDescriptor(opts.ComponentDescriptor)
	if err != nil {
		return nil, fmt.Errorf("error during serializing of the resolved components: %w", err)
	}
	components, err := serializeComponentDescriptorList(opts.ComponentDescriptors)
	if err != nil {
		return nil, fmt.Errorf("error during serializing of the component descriptor: %w", err)
	}

	values := map[string]interface{}{
		"imports":    opts.Imports,
		"cd":         component,
		"components": components,
	}

	// add blueprint and component descriptor ref information to the input values
	if opts.Installation != nil {
		blueprintDef, err := utils.JSONSerializeToGenericObject(opts.Installation.Spec.Blueprint)
		if err != nil {
			return nil, fmt.Errorf("unable to serialize the blueprint definition")
		}
		values["blueprint"] = blueprintDef

		if opts.Installation.Spec.ComponentDescriptor != nil {
			cdDef, err := utils.JSONSerializeToGenericObject(opts.Installation.Spec.ComponentDescriptor)
			if err != nil {
				return nil, fmt.Errorf("unable to serialize the component descriptor definition")
			}
			values["componentDescriptorDef"] = cdDef
		}
	}

	executionItems := lsv1alpha1.DeployItemTemplateList{}
	for _, tmplExec := range opts.Blueprint.Info.DeployExecutions {
		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		executionItemsBytes, err := impl.TemplateDeployExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, err
		}

		output := &DeployExecutorOutput{}
		if err := yaml.Unmarshal(executionItemsBytes, output); err != nil {
			return nil, fmt.Errorf("error while decoding templated execution: %w", err)
		}
		if output.DeployItems == nil {
			continue
		}
		executionItems = append(executionItems, output.DeployItems...)
	}

	if err := validateDeployItemList(field.NewPath("deployExecutions"), executionItems); err != nil {
		return nil, err
	}

	return executionItems, nil
}

func validateDeployItemList(fldPath *field.Path, list lsv1alpha1.DeployItemTemplateList) error {
	coreList := core.DeployItemTemplateList{}
	if err := lsv1alpha1.Convert_v1alpha1_DeployItemTemplateList_To_core_DeployItemTemplateList(&list, &coreList, nil); err != nil {
		return err
	}
	return validation.ValidateDeployItemTemplateList(fldPath, coreList).ToAggregate()
}

// ExportExecutorOutput describes the output of export executor.
type ExportExecutorOutput struct {
	Exports map[string]interface{} `json:"exports"`
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
		output := &ExportExecutorOutput{}
		if err := yaml.Unmarshal(exportDataJSONBytes, output); err != nil {
			return nil, err
		}
		exportData = utils.MergeMaps(exportData, output.Exports)
	}

	return exportData, nil
}

func serializeComponentDescriptor(cd *cdv2.ComponentDescriptor) (interface{}, error) {
	if cd == nil {
		return nil, nil
	}
	data, err := codec.Encode(cd)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}

func serializeComponentDescriptorList(cd *cdv2.ComponentDescriptorList) (interface{}, error) {
	if cd == nil {
		return nil, nil
	}
	data, err := codec.Encode(cd)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}
