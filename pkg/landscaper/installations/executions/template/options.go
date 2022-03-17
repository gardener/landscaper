// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/utils"
)

// BlueprintExecutionOptions describes the base options for templating of all blueprint executions.
type BlueprintExecutionOptions struct {
	Installation         *lsv1alpha1.Installation
	Blueprint            *blueprints.Blueprint
	ComponentDescriptor  *cdv2.ComponentDescriptor
	ComponentDescriptors *cdv2.ComponentDescriptorList
	Imports              map[string]interface{}
}

// NewBlueprintExecutionOptions create new basic blueprint execution options
func NewBlueprintExecutionOptions(installation *lsv1alpha1.Installation, blueprint *blueprints.Blueprint, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, imports map[string]interface{}) BlueprintExecutionOptions {
	return BlueprintExecutionOptions{
		Installation:         installation,
		Blueprint:            blueprint,
		ComponentDescriptor:  cd,
		ComponentDescriptors: cdList,
		Imports:              imports,
	}
}

func (o *BlueprintExecutionOptions) Values() (map[string]interface{}, error) {
	// marshal and unmarshal resolved component descriptor
	component, err := serializeComponentDescriptor(o.ComponentDescriptor)
	if err != nil {
		return nil, fmt.Errorf("error during serializing of the resolved components: %w", err)
	}
	components, err := serializeComponentDescriptorList(o.ComponentDescriptors)
	if err != nil {
		return nil, fmt.Errorf("error during serializing of the component descriptor: %w", err)
	}

	values := map[string]interface{}{
		"cd":         component,
		"components": components,
		"imports":    o.Imports,
	}

	// add blueprint and component descriptor ref information to the input values
	if o.Installation != nil {
		blueprintDef, err := utils.JSONSerializeToGenericObject(o.Installation.Spec.Blueprint)
		if err != nil {
			return nil, fmt.Errorf("unable to serialize the blueprint definition")
		}
		values["blueprint"] = blueprintDef
		values["blueprintDef"] = blueprintDef

		if o.Installation.Spec.ComponentDescriptor != nil {
			cdDef, err := utils.JSONSerializeToGenericObject(o.Installation.Spec.ComponentDescriptor)
			if err != nil {
				return nil, fmt.Errorf("unable to serialize the component descriptor definition")
			}
			values["componentDescriptorDef"] = cdDef
		}
	}
	return values, nil
}

////////////////////////////////////////////////////////////////////////////////

// DeployExecutionOptions describes the options for templating the deploy executions.
type DeployExecutionOptions struct {
	BlueprintExecutionOptions
}

func NewDeployExecutionOptions(base BlueprintExecutionOptions) DeployExecutionOptions {
	return DeployExecutionOptions{
		BlueprintExecutionOptions: base,
	}
}

func (o *DeployExecutionOptions) Values() (map[string]interface{}, error) {
	return o.BlueprintExecutionOptions.Values()
}

////////////////////////////////////////////////////////////////////////////////

// ExportExecutionOptions describes the options for templating the deploy executions.
type ExportExecutionOptions struct {
	BlueprintExecutionOptions
	Exports map[string]interface{}
}

func NewExportExecutionOptions(base BlueprintExecutionOptions, exports map[string]interface{}) ExportExecutionOptions {
	return ExportExecutionOptions{
		BlueprintExecutionOptions: base,
		Exports:                   exports,
	}
}

func (o *ExportExecutionOptions) Values() (map[string]interface{}, error) {
	values, err := o.BlueprintExecutionOptions.Values()
	if err != nil {
		return nil, err
	}
	values["values"] = o.Exports

	for k, v := range o.Exports {
		values[k] = v
	}
	return values, nil
}
