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
	"errors"
	"fmt"
	"github.com/gardener/landscaper/pkg/apis/core"
	"github.com/gardener/landscaper/pkg/apis/core/install"
	flag "github.com/spf13/pflag"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"os"
)

// Format of the generated configuration
type OutputFormat string

const (
	OutputFormatJson OutputFormat = "json"
	OutputFormatYaml OutputFormat = "yaml"
)

type options struct {
	FilePaths  []string
	Format     string
	OutputPath string

	outputFormat OutputFormat
	components   []*core.Blueprint
}

func NewOptions() *options {
	return &options{
		components: make([]*core.Blueprint, 0),
	}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	if fs == nil {
		fs = flag.CommandLine
	}

	fs.StringArrayVarP(&o.FilePaths, "file", "f", []string{}, "File or directory containing Components")
	fs.StringVar(&o.Format, "format", string(OutputFormatYaml), "Output format of the generated configuration")
	fs.StringVarP(&o.OutputPath, "output-dir", "o", "", "Write the contructed config yaml files to the given directory")
}

func (o *options) Complete() error {
	if len(o.FilePaths) == 0 {
		return errors.New("at least one file has to be defined")
	}

	o.outputFormat = OutputFormat(o.Format)

	if len(o.OutputPath) != 0 {
		outInfo, err := os.Stat(o.OutputPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			if err := os.MkdirAll(o.OutputPath, os.ModePerm); err != nil {
				return err
			}
		} else {
			if !outInfo.IsDir() {
				return fmt.Errorf("output path %s has to be a directory", o.OutputPath)
			}
		}
	}

	landscaperScheme := runtime.NewScheme()
	install.Install(landscaperScheme)
	decoder := serializer.NewCodecFactory(landscaperScheme).UniversalDecoder()

	for _, fp := range o.FilePaths {
		finfo, err := os.Stat(fp)
		if err != nil {
			return err
		}

		if finfo.IsDir() {
			return errors.New("currently only files are supported")
		}

		data, err := ioutil.ReadFile(fp)
		if err != nil {
			return err
		}

		component := &core.Blueprint{}
		if _, _, err := decoder.Decode(data, nil, component); err != nil {
			return err
		}

		o.components = append(o.components, component)
	}

	return nil
}
