// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/hack/generate-schemes/generators"
	lsschema "github.com/gardener/landscaper/apis/schema"
	"github.com/go-openapi/spec"
	"sigs.k8s.io/yaml"

	"github.com/go-openapi/jsonreference"

	"github.com/gardener/landscaper/apis/openapi"
)

var Exports = []string{
	"Blueprint",
	"v1alpha1.LandscaperConfiguration",
	"ProviderConfiguration",
	"ProviderStatus",
	"v1alpha1.Configuration",
}

var CRDs = []lsschema.CustomResourceDefinitions{
	lsv1alpha1.ResourceDefinition,
}

var (
	schemaDir string
	crdDir string
)

func init() {
	flag.StringVar(&schemaDir, "schema-dir", "", "output directory for jsonschemas")
	flag.StringVar(&crdDir, "crd-dir", "", "output directory for crds")
}

func main() {
	flag.Parse()
	if len(schemaDir) == 0 {
		log.Fatalln("expected --schema-dir to be set")
	}
	if err := run(schemaDir, crdDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run(schemaDir, crdDir string) error {
	if err := prepareExportDir(schemaDir); err != nil {
		return err
	}
	if err := prepareExportDir(crdDir); err != nil {
		return err
	}

	refCallback := func(path string) spec.Ref {
		ref, _ := jsonreference.New(generators.DefinitionsRef(path))
		return spec.Ref{Ref: ref}
	}
	jsonGen := &generators.JSONSchemaGenerator{
		Definitions: openapi.GetOpenAPIDefinitions(refCallback),
	}
	for defName, apiDefinition := range jsonGen.Definitions {
		if !ShouldCreateDefinition(Exports, defName) {
			continue
		}
		data, err := jsonGen.Generate(defName, apiDefinition)
		if err != nil {
			return fmt.Errorf("unable to generate jsonschema for %s: %w", defName, err)
		}

		// write file
		file := filepath.Join(schemaDir, generators.ParsePackageVersionName(defName).String() + ".json")
		if err := ioutil.WriteFile(file, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write jsonschema for %q to %q: %w", file, generators.ParsePackageVersionName(defName).String(), err)
		}

		fmt.Printf("Generated jsonschema for %q...\n", generators.ParsePackageVersionName(defName).String())
	}

	if len(crdDir) == 0 {
		log.Println("Skip crd generation")
		return nil
	}
	crdGen := generators.NewCRDGenerator(openapi.GetOpenAPIDefinitions)
	for _, crdVersion := range CRDs {
		for _, crdDef := range crdVersion.Definitions {
			if err := crdGen.Generate(crdVersion.Group, crdVersion.Version, crdDef); err != nil {
				return fmt.Errorf("unable to generate crd for %s %s %s: %w", crdVersion.Group, crdVersion.Version, crdDef.Names.Kind, err)
			}
		}
	}

	crds, err := crdGen.CRDs()
	if err != nil {
		return err
	}
	for _, crd := range crds {
		jsonBytes, err := json.Marshal(crd)
		if err != nil {
			return fmt.Errorf("unable to marshal CRD %s: %w", crd.Name, err)
		}
		data, err := yaml.JSONToYAML(jsonBytes)
		if err != nil {
			return fmt.Errorf("unable to convert json of CRD %s to yaml: %w", crd.Name, err)
		}

		// write file
		file := filepath.Join(crdDir, fmt.Sprintf("%s_%s.yaml", crd.Spec.Group, crd.Spec.Names.Plural))
		if err := ioutil.WriteFile(file, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write crd for %q to %q: %w", file, crd.Name, err)
		}

		fmt.Printf("Generated crd for %q...\n", crd.Name)
	}

	return nil
}

// ShouldCreateDefinition checks whether the definition should be exported
func ShouldCreateDefinition(exportNames []string, defName string) bool {
	for _, exportName := range exportNames {
		if strings.HasSuffix(defName, exportName) {
			return true
		}
	}
	return false
}

func prepareExportDir(exportDir string) error {
	if err := os.MkdirAll(exportDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to to create export directory %q: %w", exportDir, err)
	}
	// cleanup previous files
	files, err := ioutil.ReadDir(exportDir)
	if err != nil {
		return fmt.Errorf("unable to read files from export directory: %w", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := filepath.Join(exportDir, file.Name())
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("unable to remove %s: %w", filename, err)
		}
	}
	return nil
}
