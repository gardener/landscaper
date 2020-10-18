// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

// GoTemplateExecution is the go template implementation for landscaper templating.
type GoTemplateExecution struct {
	state GenericStateHandler
}

// GoTemplateResult describes the result of go templating.
type GoTemplateResult struct {
	State json.RawMessage `json:"state"`
}

// GoTemplate is the GoTemplate executor for a deploy execution.
func (t *GoTemplateExecution) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, components, imports interface{}) ([]byte, error) {
	var rawTemplate string
	if len(tmplExec.Template) != 0 {
		if err := json.Unmarshal(tmplExec.Template, &rawTemplate); err != nil {
			return nil, err
		}
	}
	if len(tmplExec.File) != 0 {
		rawTemplateBytes, err := vfs.ReadFile(blueprint.Fs, tmplExec.File)
		if err != nil {
			return nil, err
		}
		rawTemplate = string(rawTemplateBytes)
	}
	if len(rawTemplate) == 0 {
		return nil, fmt.Errorf("no template found")
	}

	ctx := context.Background()
	defer ctx.Done()
	state, err := t.getState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	tmpl, err := template.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs)).
		Option("missingkey=zero").
		Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"imports": imports,
		"cd":      components,
		"state":   state,
	}
	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}
	if err := t.storeState(ctx, tmplExec, data.Bytes()); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	return data.Bytes(), nil
}

func (t *GoTemplateExecution) TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) ([]byte, error) {
	var rawTemplate string
	if len(tmplExec.Template) != 0 {
		if err := json.Unmarshal(tmplExec.Template, &rawTemplate); err != nil {
			return nil, err
		}
	}
	if len(tmplExec.File) != 0 {
		rawTemplateBytes, err := vfs.ReadFile(blueprint.Fs, tmplExec.File)
		if err != nil {
			return nil, err
		}
		rawTemplate = string(rawTemplateBytes)
	}
	if len(rawTemplate) == 0 {
		return nil, fmt.Errorf("no template found")
	}

	ctx := context.Background()
	defer ctx.Done()
	state, err := t.getState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	tmpl, err := template.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs)).
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
	if err := t.storeState(ctx, tmplExec, data.Bytes()); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	return data.Bytes(), nil
}

func (t *GoTemplateExecution) getState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (interface{}, error) {
	if t.state == nil {
		return map[string]interface{}{}, nil
	}
	data, err := t.state.Get(ctx, tmplExec.Name)
	if err != nil {
		if err == StateNotFoundErr {
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

func (t *GoTemplateExecution) storeState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, data []byte) error {
	if t.state == nil {
		return nil
	}
	res := &GoTemplateResult{}
	if err := yaml.Unmarshal(data, res); err != nil {
		return err
	}
	return t.state.Store(ctx, tmplExec.Name, res.State)
}

// LandscaperSprigFuncMap returns the sanitized spring function map.
func LandscaperSprigFuncMap() template.FuncMap {
	fm := sprig.FuncMap()
	delete(fm, "env")
	delete(fm, "expandenv")
	return template.FuncMap(fm)
}

// LandscaperTplFuncMap contains all additional landscaper functions that are
// available in the executors templates.
func LandscaperTplFuncMap(fs vfs.FileSystem) map[string]interface{} {
	return map[string]interface{}{
		"readFile": readFileFunc(fs),
		"readDir":  readDir(fs),
		"toYaml":   toYAML,
	}
}

// readFileFunc returns a function that reads a file from a location in a filesystem
func readFileFunc(fs vfs.FileSystem) func(path string) []byte {
	return func(path string) []byte {
		file, err := vfs.ReadFile(fs, path)
		if err != nil {
			// maybe we should ignore the error and return an empty byte array
			panic(err)
		}
		return file
	}
}

// readDir lists all files of directory
func readDir(fs vfs.FileSystem) func(path string) []os.FileInfo {
	return func(path string) []os.FileInfo {
		files, err := vfs.ReadDir(fs, path)
		if err != nil {
			// maybe we should ignore the error and return an empty byte array
			panic(err)
		}
		return files
	}
}

// toYAML takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}
