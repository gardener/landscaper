// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package spiff

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

// Templater describes the spiff template implementation for execution templater.
type Templater struct {
	state template.GenericStateHandler
}

// New creates a new spiff execution templater.
func New(state template.GenericStateHandler) *Templater {
	return &Templater{
		state: state,
	}
}

func (t Templater) Type() lsv1alpha1.TemplateType {
	return lsv1alpha1.SpiffTemplateType
}

func (t *Templater) TemplateSubinstallationExecutions(tmplExec lsv1alpha1.TemplateExecutor,
	blueprint *blueprints.Blueprint,
	cd *cdv2.ComponentDescriptor,
	cdList *cdv2.ComponentDescriptorList,
	values map[string]interface{}) (*template.SubinstallationExecutorOutput, error) {

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
	LandscaperSpiffFuncs(functions, cd, cdList)

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

	data, err := spiffyaml.Marshal(res)
	if err != nil {
		return nil, err
	}
	output := &template.SubinstallationExecutorOutput{}
	if err := yaml.Unmarshal(data, output); err != nil {
		return nil, err
	}
	return output, nil
}

func (t *Templater) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, descriptor *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) (*template.DeployExecutorOutput, error) {
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

	data, err := spiffyaml.Marshal(res)
	if err != nil {
		return nil, err
	}
	output := &template.DeployExecutorOutput{}
	if err := yaml.Unmarshal(data, output); err != nil {
		return nil, err
	}
	return output, nil
}

func (t *Templater) TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) (*template.ExportExecutorOutput, error) {
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
	data, err := spiffyaml.Marshal(res)
	if err != nil {
		return nil, err
	}
	output := &template.ExportExecutorOutput{}
	if err := yaml.Unmarshal(data, output); err != nil {
		return nil, err
	}
	return output, nil
}

func (t *Templater) templateNode(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint) (spiffyaml.Node, error) {
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

func (t *Templater) getDeployExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (spiffyaml.Node, error) {
	return t.getState(ctx, "deploy", tmplExec)
}

func (t *Templater) storeDeployExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, spiff spiffing.Spiff, res spiffyaml.Node) error {
	return t.storeState(ctx, "deploy", tmplExec, spiff, res)
}

func (t *Templater) getExportExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (spiffyaml.Node, error) {
	return t.getState(ctx, "export", tmplExec)
}

func (t *Templater) storeExportExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, spiff spiffing.Spiff, res spiffyaml.Node) error {
	return t.storeState(ctx, "export", tmplExec, spiff, res)
}

func (t *Templater) getState(ctx context.Context, prefix string, tmplExec lsv1alpha1.TemplateExecutor) (spiffyaml.Node, error) {
	if t.state == nil {
		return spiffyaml.NewNode(map[string]interface{}{}, "state"), nil
	}
	stateBytes, err := t.state.Get(ctx, prefix+tmplExec.Name)
	if err != nil {
		if err != template.StateNotFoundErr {
			return spiffyaml.NewNode(map[string]interface{}{}, "state"), nil
		}
	}
	return spiffyaml.Unmarshal("stateHdl", stateBytes)
}

func (t *Templater) storeState(ctx context.Context, prefix string, tmplExec lsv1alpha1.TemplateExecutor, spiff spiffing.Spiff, res spiffyaml.Node) error {
	if t.state == nil {
		return nil
	}
	stateBytes, err := spiffyaml.Marshal(spiff.DetermineState(res))
	if err != nil {
		return fmt.Errorf("unable to marshal state: %w", err)
	}
	if len(stateBytes) == 0 {
		return nil
	}

	if err := t.state.Store(ctx, prefix+tmplExec.Name, stateBytes); err != nil {
		return fmt.Errorf("unabel to persists state: %w", err)
	}
	return nil
}
