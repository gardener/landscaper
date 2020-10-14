// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
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
type GoTemplateExecution struct{}

// GoTemplate is the GoTemplate executor for a deploy execution.
func (_ *GoTemplateExecution) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, components, imports interface{}) ([]byte, error) {
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

	tmpl, err := template.New("execution").
		Funcs(template.FuncMap(sprig.FuncMap())).Funcs(LandscaperTplFuncMap(blueprint.Fs)).
		Option("missingkey=zero").
		Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"imports": imports,
		"cd":      components,
	}
	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

func (_ *GoTemplateExecution) TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) ([]byte, error) {
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

	// we only start with go template + sprig
	tmpl, err := template.New("execution").Funcs(template.FuncMap(sprig.FuncMap())).Funcs(LandscaperTplFuncMap(blueprint.Fs)).Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"exports": exports,
	}
	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
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
