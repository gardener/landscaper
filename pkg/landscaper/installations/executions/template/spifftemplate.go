// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"github.com/mandelsoft/vfs/pkg/vfs"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

type SpiffTemplate struct {
	state GenericStateHandler
}

func (t *SpiffTemplate) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, components, imports interface{}) ([]byte, error) {
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

	values := map[string]interface{}{
		"imports": imports,
		"cd":      components,
	}

	spiff, err := spiffing.New().WithFunctions(spiffing.NewFunctions()).WithFileSystem(blueprint.Fs).WithValues(values)
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
	if len(tmplExec.Template) != 0 {
		return spiffyaml.Unmarshal("template", tmplExec.Template)
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
