// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	gotmpl "text/template"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	lstmpl "github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

// Templater is the go template implementation for landscaper templating.
type Templater struct {
	blobResolver ctf.BlobResolver
	state        lstmpl.GenericStateHandler
}

// New creates a new go template execution templater.
func New(blobResolver ctf.BlobResolver, state lstmpl.GenericStateHandler) *Templater {
	return &Templater{
		blobResolver: blobResolver,
		state:        state,
	}
}

// StateTemplateResult describes the result of go templating.
type StateTemplateResult struct {
	State json.RawMessage `json:"state"`
}

func (t Templater) Type() lsv1alpha1.TemplateType {
	return lsv1alpha1.GOTemplateType
}

func (t *Templater) TemplateSubinstallationExecutions(tmplExec lsv1alpha1.TemplateExecutor,
	blueprint *blueprints.Blueprint,
	cd *cdv2.ComponentDescriptor,
	cdList *cdv2.ComponentDescriptorList,
	values map[string]interface{}) (*lstmpl.SubinstallationExecutorOutput, error) {

	rawTemplate, err := getTemplateFromExecution(tmplExec, blueprint)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	defer ctx.Done()
	state, err := t.getDeployExecutionState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	tmpl, err := gotmpl.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs, cd, cdList, t.blobResolver)).
		Option("missingkey=zero").
		Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	values["state"] = state
	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}
	if err := t.storeDeployExecutionState(ctx, tmplExec, data.Bytes()); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	output := &lstmpl.SubinstallationExecutorOutput{}
	if err := yaml.Unmarshal(data.Bytes(), output); err != nil {
		return nil, fmt.Errorf("error while decoding templated execution: %w", err)
	}
	return output, nil
}

// TemplateDeployExecutions is the GoTemplate executor for a deploy execution.
func (t *Templater) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint,
	descriptor *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) (*lstmpl.DeployExecutorOutput, error) {
	rawTemplate, err := getTemplateFromExecution(tmplExec, blueprint)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	defer ctx.Done()
	state, err := t.getDeployExecutionState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	tmpl, err := gotmpl.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs, descriptor, cdList, t.blobResolver)).
		Option("missingkey=zero").
		Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	values["state"] = state
	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}
	if err := t.storeDeployExecutionState(ctx, tmplExec, data.Bytes()); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	output := &lstmpl.DeployExecutorOutput{}
	if err := yaml.Unmarshal(data.Bytes(), output); err != nil {
		return nil, fmt.Errorf("error while decoding templated execution: %w", err)
	}
	return output, nil
}

func (t *Templater) TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) (*lstmpl.ExportExecutorOutput, error) {
	rawTemplate, err := getTemplateFromExecution(tmplExec, blueprint)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	defer ctx.Done()
	state, err := t.getExportExecutionState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	tmpl, err := gotmpl.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs, nil, nil, t.blobResolver)).
		Option("missingkey=zero").
		Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"values": exports,
		"state":  state,
	}
	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}
	if err := t.storeExportExecutionState(ctx, tmplExec, data.Bytes()); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	output := &lstmpl.ExportExecutorOutput{}
	if err := yaml.Unmarshal(data.Bytes(), output); err != nil {
		return nil, err
	}
	return output, nil
}

func (t *Templater) getDeployExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (interface{}, error) {
	return t.getState(ctx, "deploy", tmplExec)
}

func (t *Templater) storeDeployExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, data []byte) error {
	return t.storeState(ctx, "deploy", tmplExec, data)
}

func (t *Templater) getExportExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (interface{}, error) {
	return t.getState(ctx, "export", tmplExec)
}

func (t *Templater) storeExportExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, data []byte) error {
	return t.storeState(ctx, "export", tmplExec, data)
}

func (t *Templater) getState(ctx context.Context, prefix string, tmplExec lsv1alpha1.TemplateExecutor) (interface{}, error) {
	if t.state == nil {
		return map[string]interface{}{}, nil
	}
	data, err := t.state.Get(ctx, prefix+tmplExec.Name)
	if err != nil {
		if err == lstmpl.StateNotFoundErr {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}

	var state interface{}
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func (t *Templater) storeState(ctx context.Context, prefix string, tmplExec lsv1alpha1.TemplateExecutor, data []byte) error {
	if t.state == nil {
		return nil
	}
	res := &StateTemplateResult{}
	if err := yaml.Unmarshal(data, res); err != nil {
		return err
	}
	if len(res.State) == 0 {
		return nil
	}
	return t.state.Store(ctx, prefix+tmplExec.Name, res.State)
}

func getTemplateFromExecution(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint) (string, error) {
	if len(tmplExec.Template.RawMessage) != 0 {
		var rawTemplate string
		if err := json.Unmarshal(tmplExec.Template.RawMessage, &rawTemplate); err != nil {
			return "", err
		}
		return rawTemplate, nil
	}
	if len(tmplExec.File) != 0 {
		rawTemplateBytes, err := vfs.ReadFile(blueprint.Fs, tmplExec.File)
		if err != nil {
			return "", err
		}
		return string(rawTemplateBytes), nil
	}
	return "", fmt.Errorf("no template found")
}
