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
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	lstmpl "github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

const (
	recursionMaxNums = 100
)

// Templater is the go template implementation for landscaper templating.
type Templater struct {
	blobResolver   ctf.BlobResolver
	state          lstmpl.GenericStateHandler
	inputFormatter *lstmpl.TemplateInputFormatter
}

// New creates a new go template execution templater.
func New(blobResolver ctf.BlobResolver, state lstmpl.GenericStateHandler) *Templater {
	return &Templater{
		blobResolver:   blobResolver,
		state:          state,
		inputFormatter: lstmpl.NewTemplateInputFormatter(false, "imports", "values", "state"),
	}
}

// WithInputFormatter ads a custom input formatter to this templater used for error messages.
func (t *Templater) WithInputFormatter(inputFormatter *lstmpl.TemplateInputFormatter) *Templater {
	t.inputFormatter = inputFormatter
	return t
}

type TemplateExecution struct {
	funcMap       map[string]interface{}
	blueprint     *blueprints.Blueprint
	includedNames map[string]int
}

func NewTemplateExecution(blueprint *blueprints.Blueprint, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, blobResolver ctf.BlobResolver) *TemplateExecution {
	t := &TemplateExecution{
		funcMap:       LandscaperTplFuncMap(blueprint.Fs, cd, cdList, blobResolver),
		blueprint:     blueprint,
		includedNames: map[string]int{},
	}
	t.funcMap["include"] = t.include
	return t
}

func (te *TemplateExecution) include(name string, binding interface{}) (string, error) {
	if v, ok := te.includedNames[name]; ok {
		if v > recursionMaxNums {
			return "", errors.Wrapf(fmt.Errorf("unable to execute template"), "rendering template has a nested reference name: %s", name)
		}
		te.includedNames[name]++
	} else {
		te.includedNames[name] = 1
	}
	data, err := vfs.ReadFile(te.blueprint.Fs, name)
	if err != nil {
		return "", errors.Wrapf(err, "unable to read include file %q", name)
	}
	res, err := te.Execute(string(data), binding)
	te.includedNames[name]--
	return string(res), err
}

func (te *TemplateExecution) Execute(template string, binding interface{}) ([]byte, error) {
	tmpl, err := gotmpl.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(te.funcMap).
		Option("missingkey=zero").
		Parse(template)
	if err != nil {
		parseError := TemplateErrorBuilder(err).WithSource(&template).Build()
		return nil, parseError
	}

	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, binding); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

// StateTemplateResult describes the result of go templating.
type StateTemplateResult struct {
	State json.RawMessage `json:"state"`
}

func (t Templater) Type() lsv1alpha1.TemplateType {
	return lsv1alpha1.GOTemplateType
}

func (t *Templater) TemplateExecution(rawTemplate string, blueprint *blueprints.Blueprint, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) ([]byte, error) {
	te := NewTemplateExecution(blueprint, cd, cdList, t.blobResolver)
	return te.Execute(rawTemplate, values)
}

func (t *Templater) TemplateSubinstallationExecutions(tmplExec lsv1alpha1.TemplateExecutor,
	blueprint *blueprints.Blueprint,
	cd *cdv2.ComponentDescriptor,
	cdList *cdv2.ComponentDescriptorList,
	values map[string]interface{}) (*lstmpl.SubinstallationExecutorOutput, error) {

	const templateName = "subinstallation execution"

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

	values["state"] = state
	data, err := t.TemplateExecution(rawTemplate, blueprint, cd, cdList, values)
	if err != nil {
		executeError := TemplateErrorBuilder(err).WithSource(&rawTemplate).
			WithInput(values, t.inputFormatter).
			Build()
		return nil, executeError
	}

	if err := CreateErrorIfContainsNoValue(string(data), templateName, values, t.inputFormatter); err != nil {
		return nil, err
	}

	if err := t.storeDeployExecutionState(ctx, tmplExec, data); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	output := &lstmpl.SubinstallationExecutorOutput{}
	if err := yaml.Unmarshal(data, output); err != nil {
		return nil, fmt.Errorf("error while decoding templated execution: %w", err)
	}
	return output, nil
}

// TemplateImportExecutions is the GoTemplate executor for an import execution.
func (t *Templater) TemplateImportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint,
	descriptor *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) (*lstmpl.ImportExecutorOutput, error) {
	rawTemplate, err := getTemplateFromExecution(tmplExec, blueprint)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	defer ctx.Done()

	data, err := t.TemplateExecution(rawTemplate, blueprint, descriptor, cdList, values)
	if err != nil {
		executeError := TemplateErrorBuilder(err).WithSource(&rawTemplate).
			WithInput(values, t.inputFormatter).
			Build()
		return nil, executeError
	}

	output := &lstmpl.ImportExecutorOutput{}
	if err := yaml.Unmarshal(data, output); err != nil {
		return nil, fmt.Errorf("error while decoding templated execution: %w", err)
	}
	return output, nil
}

// TemplateDeployExecutions is the GoTemplate executor for a deploy execution.
func (t *Templater) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint,
	descriptor *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) (*lstmpl.DeployExecutorOutput, error) {

	const templateName = "deploy execution"

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

	values["state"] = state
	data, err := t.TemplateExecution(rawTemplate, blueprint, descriptor, cdList, values)
	if err != nil {
		executeError := TemplateErrorBuilder(err).WithSource(&rawTemplate).
			WithInput(values, t.inputFormatter).
			Build()
		return nil, executeError
	}

	if err := CreateErrorIfContainsNoValue(string(data), templateName, values, t.inputFormatter); err != nil {
		return nil, err
	}

	if err := t.storeDeployExecutionState(ctx, tmplExec, data); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	output := &lstmpl.DeployExecutorOutput{}
	if err := yaml.Unmarshal(data, output); err != nil {
		return nil, fmt.Errorf("error while decoding templated execution: %w", err)
	}
	return output, nil
}

func (t *Templater) TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint,
	descriptor *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) (*lstmpl.ExportExecutorOutput, error) {
	const templateName = "export execution"
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

	values["state"] = state
	data, err := t.TemplateExecution(rawTemplate, blueprint, descriptor, cdList, values)
	if err != nil {
		executeError := TemplateErrorBuilder(err).WithSource(&rawTemplate).
			WithInput(values, t.inputFormatter).
			Build()
		return nil, executeError
	}

	if err := CreateErrorIfContainsNoValue(string(data), templateName, values, t.inputFormatter); err != nil {
		return nil, err
	}

	if err != nil {
		return nil, fmt.Errorf("unable to execute template: %w", err)
	}
	if err := t.storeExportExecutionState(ctx, tmplExec, data); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	output := &lstmpl.ExportExecutorOutput{}
	if err := yaml.Unmarshal(data, output); err != nil {
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
