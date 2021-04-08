// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gardener/component-cli/pkg/imagevector"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

type SpiffTemplate struct {
	state GenericStateHandler
}

func (t *SpiffTemplate) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, descriptor *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) ([]byte, error) {
	rawTemplate, err := t.templateNode(tmplExec, blueprint)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	defer ctx.Done()
	stateNode, err := t.getDeployExecutionState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	functions := spiffing.NewFunctions()
	LandscaperSpiffFuncs(functions, descriptor, cdList)

	spiff, err := spiffing.New().WithFunctions(functions).WithFileSystem(blueprint.Fs).WithValues(values)
	if err != nil {
		return nil, fmt.Errorf("unable to init spiff templater: %w", err)
	}

	res, err := spiff.Cascade(rawTemplate, nil, stateNode)
	if err != nil {
		return nil, err
	}
	if err := t.storeDeployExecutionState(ctx, tmplExec, spiff, res); err != nil {
		return nil, err
	}
	return spiffyaml.Marshal(res)
}

func (t *SpiffTemplate) TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) ([]byte, error) {
	rawTemplate, err := t.templateNode(tmplExec, blueprint)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	defer ctx.Done()
	stateNode, err := t.getExportExecutionState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	values := map[string]interface{}{
		"values": exports,
	}
	spiff, err := spiffing.New().WithFileSystem(blueprint.Fs).WithValues(values)
	if err != nil {
		return nil, fmt.Errorf("unable to init spiff templater: %w", err)
	}

	res, err := spiff.Cascade(rawTemplate, nil, stateNode)
	if err != nil {
		return nil, err
	}

	if err := t.storeExportExecutionState(ctx, tmplExec, spiff, res); err != nil {
		return nil, err
	}
	return spiffyaml.Marshal(res)
}

func (t *SpiffTemplate) templateNode(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint) (spiffyaml.Node, error) {
	if len(tmplExec.Template.RawMessage) != 0 {
		return spiffyaml.Unmarshal("template", tmplExec.Template.RawMessage)
	}
	if len(tmplExec.File) != 0 {
		rawTemplateBytes, err := vfs.ReadFile(blueprint.Fs, tmplExec.File)
		if err != nil {
			return nil, err
		}
		return spiffyaml.Unmarshal("template", rawTemplateBytes)
	}
	return nil, fmt.Errorf("no template found")
}

func (t *SpiffTemplate) getDeployExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (spiffyaml.Node, error) {
	return t.getState(ctx, "deploy", tmplExec)
}

func (t *SpiffTemplate) storeDeployExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, spiff spiffing.Spiff, res spiffyaml.Node) error {
	return t.storeState(ctx, "deploy", tmplExec, spiff, res)
}

func (t *SpiffTemplate) getExportExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (spiffyaml.Node, error) {
	return t.getState(ctx, "export", tmplExec)
}

func (t *SpiffTemplate) storeExportExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, spiff spiffing.Spiff, res spiffyaml.Node) error {
	return t.storeState(ctx, "export", tmplExec, spiff, res)
}

func (t *SpiffTemplate) getState(ctx context.Context, prefix string, tmplExec lsv1alpha1.TemplateExecutor) (spiffyaml.Node, error) {
	if t.state == nil {
		return spiffyaml.NewNode(map[string]interface{}{}, "state"), nil
	}
	stateBytes, err := t.state.Get(ctx, prefix+tmplExec.Name)
	if err != nil {
		if err != StateNotFoundErr {
			return spiffyaml.NewNode(map[string]interface{}{}, "state"), nil
		}
	}
	return spiffyaml.Unmarshal("stateHdl", stateBytes)
}

func (t *SpiffTemplate) storeState(ctx context.Context, prefix string, tmplExec lsv1alpha1.TemplateExecutor, spiff spiffing.Spiff, res spiffyaml.Node) error {
	if t.state == nil {
		return nil
	}
	stateBytes, err := spiffyaml.Marshal(spiff.DetermineState(res))
	if err != nil {
		return fmt.Errorf("unable to marshal state: %w", err)
	}

	if err := t.state.Store(ctx, prefix+tmplExec.Name, stateBytes); err != nil {
		return fmt.Errorf("unabel to persists state: %w", err)
	}
	return nil
}

func LandscaperSpiffFuncs(functions spiffing.Functions, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList) {
	functions.RegisterFunction("getResource", spiffResolveResources(cd))
	functions.RegisterFunction("getComponent", spiffResolveComponent(cd, cdList))
	functions.RegisterFunction("generateImageOverwrite", spiffGenerateImageOverwrite(cd, cdList))
}

func spiffResolveResources(cd *cdv2.ComponentDescriptor) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()
		data, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments, ""))
		if err != nil {
			return info.Error(err.Error())
		}
		var val []interface{}
		if err := yaml.Unmarshal(data, &val); err != nil {
			return info.Error(err.Error())
		}

		resources, err := resolveResources(cd, val)
		if err != nil {
			return info.Error(err.Error())
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err = json.Marshal(resources[0])
		if err != nil {
			return info.Error(err.Error())
		}

		node, err := spiffyaml.Parse("", data)
		if err != nil {
			return info.Error(err.Error())
		}
		result, err := binding.Flow(node, false)
		if err != nil {
			return info.Error(err.Error())
		}

		return result.Value(), info, true
	}
}

func spiffResolveComponent(cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()
		data, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments, ""))
		if err != nil {
			return info.Error(err.Error())
		}
		var val []interface{}
		if err := yaml.Unmarshal(data, &val); err != nil {
			return info.Error(err.Error())
		}

		components, err := resolveComponents(cd, cdList, val)
		if err != nil {
			return info.Error(err.Error())
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err = json.Marshal(components[0])
		if err != nil {
			return info.Error(err.Error())
		}

		node, err := spiffyaml.Parse("", data)
		if err != nil {
			return info.Error(err.Error())
		}
		result, err := binding.Flow(node, false)
		if err != nil {
			return info.Error(err.Error())
		}

		return result.Value(), info, true
	}
}

func spiffGenerateImageOverwrite(cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()

		internalCd := cd
		internalComponents := cdList

		if len(arguments) > 2 {
			return info.Error("Too many arguments for generateImageOverwrite.")
		}

		if len(arguments) >= 1 {
			data, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments[0], ""))
			if err != nil {
				return info.Error(err.Error())
			}

			internalCd = &cdv2.ComponentDescriptor{}
			if err := yaml.Unmarshal(data, internalCd); err != nil {
				return info.Error(err.Error())
			}
		}

		if len(arguments) == 2 {
			componentsData, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments[1], ""))
			if err != nil {
				return info.Error(err.Error())
			}

			internalComponents = &cdv2.ComponentDescriptorList{}
			if err := yaml.Unmarshal(componentsData, internalComponents); err != nil {
				return info.Error(err.Error())
			}
		}

		if internalCd == nil {
			return info.Error("No component descriptor is defined.")
		}

		if internalComponents == nil {
			return info.Error("No component descriptor list is defined.")
		}

		vector, err := imagevector.GenerateImageOverwrite(internalCd, internalComponents)
		if err != nil {
			return info.Error(err.Error())
		}

		data, err := yaml.Marshal(vector)
		if err != nil {
			return info.Error(err.Error())
		}

		node, err := spiffyaml.Parse("", data)
		if err != nil {
			return info.Error(err.Error())
		}

		result, err := binding.Flow(node, false)
		if err != nil {
			return info.Error(err.Error())
		}

		return result.Value(), info, true
	}
}
