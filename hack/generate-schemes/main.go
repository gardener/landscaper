// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/alecthomas/jsonschema"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run() error {
	gen := &JSONSchemGenerator{rootPath: ".schemas"}
	types := []GenType{
		{prefix: "landscaper_", obj: lsv1alpha1.Blueprint{}},
		{prefix: "landscaper_", obj: lsv1alpha1.InstallationTemplate{}},
		{prefix: "landscaper_", obj: config.LandscaperConfiguration{}},

		{prefix: "container_", obj: containerv1alpha1.Configuration{}},
		{prefix: "container_", obj: containerv1alpha1.ProviderConfiguration{}},
		{prefix: "container_", obj: containerv1alpha1.ProviderStatus{}},

		{prefix: "helm_", obj: helmv1alpha1.Configuration{}},
		{prefix: "helm_", obj: helmv1alpha1.ProviderConfiguration{}},
		{prefix: "helm_", obj: helmv1alpha1.ProviderStatus{}},
	}

	for _, t := range types {
		if err := gen.Generate(t); err != nil {
			return fmt.Errorf("unable to genereate schema for %s", reflect.TypeOf(t).Name())
		}
	}
	return nil
}

type JSONSchemGenerator struct {
	rootPath string
}

type GenType struct {
	prefix string
	obj interface{}
}

func (g JSONSchemGenerator) Generate(gt GenType) error {
	t := reflect.TypeOf(gt.obj)
	typeName := t.Name()
	schema := jsonschema.Reflect(gt.obj)

	if err := os.MkdirAll(g.rootPath, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create schemas directory: %w", err)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal jsonschema for %s: %w", typeName, err)
	}

	if err := ioutil.WriteFile(filepath.Join(g.rootPath, fmt.Sprintf("%s%s.json", gt.prefix, typeName)), data, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write schema json filke: %w", err)
	}
	return nil
}

