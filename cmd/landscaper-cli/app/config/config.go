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

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gardener/landscaper/pkg/apis/core"
	"github.com/gardener/landscaper/pkg/configuration/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path"
	"sigs.k8s.io/yaml"
)

func NewConfigCommand(ctx context.Context) *cobra.Command {
	opts := NewOptions()
	cmd := &cobra.Command{
		Use:   "config",
		Short: "shows the import/export configuration for the given components",

		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(); err != nil {
				fmt.Print(err.Error())
				os.Exit(1)
			}
			if err := opts.run(args); err != nil {
				fmt.Print(err.Error())
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(args []string) error {

	for _, c := range o.components {
		importInternal, importExternal, err := parseImports(c)
		if err != nil {
			return errors.Wrapf(err, "unable to parse imports for %s", c.Name)
		}
		importInternalYAML, err := formatConfiguration(o.outputFormat, importInternal)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal imports internal config for %s", c.Name)
		}
		importExternalYAML, err := formatConfiguration(o.outputFormat, importExternal)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal imports external config for %s", c.Name)
		}

		exportInternal, exportExternal, err := parseExports(c)
		if err != nil {
			return errors.Wrapf(err, "unable to parse imports for %s", c.Name)
		}
		exportInternalYAML, err := formatConfiguration(o.outputFormat, exportInternal)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal exports internal config for %s", c.Name)
		}
		exportExternalYAML, err := formatConfiguration(o.outputFormat, exportExternal)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal exports external config for %s", c.Name)
		}

		if err := o.out(c.Name, importInternalYAML, importExternalYAML, exportInternalYAML, exportExternalYAML); err != nil {
			return err
		}
	}

	return nil
}

func (o *options) out(name string, importInternal, importExternal, exportInternal, exportExternal []byte) error {
	if len(o.OutputPath) == 0 {
		fmt.Printf(":------Component %s ------:\n\n", name)
		fmt.Print(":------ Imports ------:\n\n")
		fmt.Print(":--- Internal ---:\n")
		fmt.Print(string(importInternal))
		fmt.Print(":--- External ---:\n")
		fmt.Print(string(importExternal))
		fmt.Print(":------ Exports ------:\n\n")
		fmt.Print(":--- Internal ---:\n")
		fmt.Print(string(exportInternal))
		fmt.Print(":--- External ---:\n")
		fmt.Print(string(exportExternal))
	}

	// write to file
	importsInternalFileName, err := formatFileName(o.outputFormat, fmt.Sprintf("%s-imports-internal", name))
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(o.OutputPath, importsInternalFileName), importInternal, os.ModePerm); err != nil {
		return err
	}

	importsExternalFileName, err := formatFileName(o.outputFormat, fmt.Sprintf("%s-imports-external", name))
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(o.OutputPath, importsExternalFileName), importExternal, os.ModePerm); err != nil {
		return err
	}

	exportsInternalFileName, err := formatFileName(o.outputFormat, fmt.Sprintf("%s-exports-internal", name))
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(o.OutputPath, exportsInternalFileName), exportInternal, os.ModePerm); err != nil {
		return err
	}

	exportsExternalFileName, err := formatFileName(o.outputFormat, fmt.Sprintf("%s-exports-external", name))
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(o.OutputPath, exportsExternalFileName), importInternal, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func formatFileName(format OutputFormat, name string) (string, error) {
	switch format {
	case OutputFormatYaml:
		return fmt.Sprintf("%s.yaml", name), nil
	case OutputFormatJson:
		return fmt.Sprintf("%s.json", name), nil
	default:
		return "", errors.Errorf("unsupported format %s", format)
	}
}

func formatConfiguration(format OutputFormat, obj interface{}) ([]byte, error) {
	switch format {
	case OutputFormatYaml:
		data, err := yaml.Marshal(obj)
		if err != nil {
			return nil, errors.Wrap(err, "unable to marshal config to yaml")
		}
		return data, nil
	case OutputFormatJson:
		data, err := json.Marshal(obj)
		if err != nil {
			return nil, errors.Wrap(err, "unable to marshal config to json")
		}
		return data, nil
	default:
		return nil, errors.Errorf("unsupported format %s", format)
	}
}

func parseImports(component *core.Component) (map[string]interface{}, map[string]interface{}, error) {
	internalConfig := make(map[string]interface{})
	externalConfig := make(map[string]interface{})
	for _, imp := range component.Spec.Imports {
		// format the jsonpath to internal parsable path
		toPath := fmt.Sprintf("{%s}", imp.To)
		cfg, err := jsonpath.Construct(toPath)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unable to construct config at to: %s", imp.To)
		}
		internalConfig = utils.MergeMaps(internalConfig, cfg)

		// format the jsonpath to internal parsable path
		fromPath := fmt.Sprintf("{%s}", imp.From)
		cfg, err = jsonpath.Construct(fromPath)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unable to construct config at to: %s", imp.From)
		}
		externalConfig = utils.MergeMaps(externalConfig, cfg)
	}
	return internalConfig, externalConfig, nil
}

func parseExports(component *core.Component) (map[string]interface{}, map[string]interface{}, error) {
	internalConfig := make(map[string]interface{})
	externalConfig := make(map[string]interface{})
	for _, exp := range component.Spec.Exports {
		// format the jsonpath to internal parsable path
		toPath := fmt.Sprintf("{%s}", exp.To)
		cfg, err := jsonpath.Construct(toPath)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unable to construct config at to: %s", exp.To)
		}
		externalConfig = utils.MergeMaps(externalConfig, cfg)

		// format the jsonpath to internal parsable path
		fromPath := fmt.Sprintf("{%s}", exp.From)
		cfg, err = jsonpath.Construct(fromPath)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unable to construct config at to: %s", exp.From)
		}
		internalConfig = utils.MergeMaps(internalConfig, cfg)
	}
	return internalConfig, externalConfig, nil
}
