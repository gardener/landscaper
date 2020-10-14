// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"github.com/mandelsoft/vfs/pkg/vfs"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

type SpiffTemplate struct{}

func (t *SpiffTemplate) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, components, imports interface{}) ([]byte, error) {
	rawTemplate, err := t.templateNode(tmplExec, blueprint)
	if err != nil {
		return nil, err
	}
	values := map[string]interface{}{
		"imports": imports,
		"cd":      components,
	}

	spiff, err := spiffing.New().WithFunctions(spiffing.NewFunctions()).WithFileSystem(blueprint.Fs).WithValues(values)
	if err != nil {
		return nil, fmt.Errorf("unable to init spiff templater: %w", err)
	}

	res, err := spiff.Cascade(rawTemplate, nil)
	if err != nil {
		return nil, err
	}
	return spiffyaml.Marshal(res)
}

func (t *SpiffTemplate) TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) ([]byte, error) {
	rawTemplate, err := t.templateNode(tmplExec, blueprint)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"exports": exports,
	}
	spiff, err := spiffing.New().WithFunctions(spiffing.NewFunctions()).WithFileSystem(blueprint.Fs).WithValues(values)
	if err != nil {
		return nil, fmt.Errorf("unable to init spiff templater: %w", err)
	}

	res, err := spiff.Cascade(rawTemplate, nil)
	if err != nil {
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
