// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"

	"github.com/Masterminds/sprig/v3"
	"github.com/mandelsoft/vfs/pkg/vfs"

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

	// we only start with go template + sprig
	tmpl, err := template.New("execution").Funcs(sprig.FuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs)).Parse(rawTemplate)
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
	tmpl, err := template.New("execution").Funcs(sprig.FuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs)).Parse(rawTemplate)
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
