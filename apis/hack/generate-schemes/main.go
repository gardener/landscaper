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
	"strings"

	"github.com/go-openapi/spec"
	"k8s.io/kube-openapi/pkg/common"
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

var CRDs = []string{
	"v1alpha1.Installation",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("expected one argument")
		os.Exit(1)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run(exportDir string) error {
	if err := prepareExportDir(exportDir); err != nil {
		return err
	}

	var definitions map[string]common.OpenAPIDefinition
	refCallback := func(path string) spec.Ref {
		ref, _ := jsonreference.New(definitionsRef(path))
		return spec.Ref{Ref: ref}
	}
	definitions = openapi.GetOpenAPIDefinitions(refCallback)

	for defName, apiDefinition := range definitions {
		if !ShouldCreateDefinition(Exports, defName) {
			continue
		}
		data, err := generateJsonSchema(defName, apiDefinition, definitions)
		if err != nil {
			return fmt.Errorf("unable to generate jsonschema for %s: %w", defName, err)
		}

		// write file
		file := filepath.Join(exportDir, PackageVersionName(defName) + ".json")
		if err := ioutil.WriteFile(file, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write jsonschema for %q to %q: %w", file, PackageVersionName(defName), err)
		}

		fmt.Printf("Generated jsonschema for %q...\n", PackageVersionName(defName))
	}


	origRefCallback := func(path string) spec.Ref {
		fmt.Println(path)
		return spec.MustCreateRef(path)
	}
	definitions = openapi.GetOpenAPIDefinitions(origRefCallback)
	for defName, apiDefinition := range definitions {
		if !ShouldCreateDefinition(CRDs, defName) {
			continue
		}
		data, err := generateCRDYamlSchema(defName, apiDefinition, definitions)
		if err != nil {
			return fmt.Errorf("unable to generate jsonschema for %s: %w", defName, err)
		}

		// write file
		file := filepath.Join(exportDir, PackageVersionName(defName) + ".yaml")
		if err := ioutil.WriteFile(file, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write jsonschema for %q to %q: %w", file, PackageVersionName(defName), err)
		}

		fmt.Printf("Generated crd for %q...\n", PackageVersionName(defName))
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

// DefinitionsPropName is the name of property where the definition is located
const DefinitionsPropName = "definitions"

func generateJsonSchema(objName string, def common.OpenAPIDefinition, definitions map[string]common.OpenAPIDefinition) ([]byte, error) {
	pvn := PackageVersionName(objName)
	// parse schema into map and add schema metadata information as well as references
	jsonSchema := make(map[string]interface{})
	data, err := json.Marshal(def.Schema)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal jsonschema: %w", err)
	}
	if err := json.Unmarshal(data, &jsonSchema); err != nil {
		return nil, fmt.Errorf("unable to decode jsonschema: %w", err)
	}

	jsonSchema["$schema"] = "https://json-schema.org/draft-07/schema#"
	jsonSchema["title"] = pvn

	dependencyDefinitions := make(map[string]spec.Schema, 0)
	if err := addDependencies(dependencyDefinitions, objName, definitions); err != nil {
		return nil, fmt.Errorf("unable to add dependencies: %w", err)
	}
	jsonSchema[DefinitionsPropName] = dependencyDefinitions
	return json.MarshalIndent(jsonSchema, "", "  ")
}

// addDependencies adds all dependencies of the given definition to the map of schemas
func addDependencies(schemas map[string]spec.Schema, defName string, definitions map[string]common.OpenAPIDefinition) error {
	def, ok := definitions[defName]
	if !ok {
		return fmt.Errorf("definition %q is not defined", defName)
	}
	for _, depName := range def.Dependencies {
		if _, ok :=schemas[PackageVersionName(depName)]; ok {
			continue
		}
		def, ok := definitions[depName]
		if !ok {
			return fmt.Errorf("dependency %q of %q cannot be found in definitions", depName, defName)
		}
		schemas[PackageVersionName(depName)] = def.Schema
		// add dependencies for that component if not already defined
		if err := addDependencies(schemas, depName, definitions); err != nil {
			return err
		}
	}
	return nil
}

// PackageVersionName parses the name from the openapi definition.
// name is expected to be some/path/<package>/<version>.<name>
func PackageVersionName(name string) string {
	splitName := strings.Split(name, "/")
	if len(splitName) < 2 {
		panic(fmt.Errorf("a component name must consits of at least a package identifier and a name.version"))
	}
	verioneName := splitName[len(splitName) - 1]
	packageName := splitName[len(splitName) - 2]
	return fmt.Sprintf("%s-%s", packageName, strings.Replace(verioneName, ".", "-", 1))
}

// definitionsRef returns the reference to the resource of a specific object name
func definitionsRef(name string) string {
	pvn := PackageVersionName(name)
	return fmt.Sprintf("#/%s/%s", DefinitionsPropName, pvn)
}

func prepareExportDir(exportDir string) error {
	if err := os.MkdirAll(exportDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to to create export directory %q: %w", exportDir, err)
	}
	// cleanup previoud files
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

func generateCRDYamlSchema(objName string, def common.OpenAPIDefinition, definitions map[string]common.OpenAPIDefinition) ([]byte, error) {
	pvn := PackageVersionName(objName)
	if err := inlineDependencies(&def, definitions); err != nil {
		return nil, fmt.Errorf("unable to add dependencies: %w", err)
	}
	// parse schema into map and add schema metadata information as well as references
	jsonSchema := make(map[string]interface{})
	data, err := json.Marshal(def.Schema)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal jsonschema: %w", err)
	}
	if err := json.Unmarshal(data, &jsonSchema); err != nil {
		return nil, fmt.Errorf("unable to decode jsonschema: %w", err)
	}

	jsonSchema["$schema"] = "https://json-schema.org/draft-07/schema#"
	jsonSchema["title"] = pvn
	jsonBytes, err := json.MarshalIndent(jsonSchema, "", "  ")
	if err != nil {
		return nil, err
	}
	return yaml.JSONToYAML(jsonBytes)
}

// inlineDependencies adds all dependencies of the given definition to the map of schemas
func inlineDependencies(def *common.OpenAPIDefinition, definitions map[string]common.OpenAPIDefinition) error {
	if len(def.Schema.Ref.String()) != 0 {
		// inline ref
		refDef, ok := definitions[def.Schema.Ref.String()]
		if !ok {
			return fmt.Errorf("dependency %q cannot be found in definitions", def.Schema.Ref.String())
		}
		if err := inlineDependencies(&refDef, definitions); err != nil {
			return fmt.Errorf("dependency %q cannot be inlined: %w", def.Schema.Ref.String(), err)
		}
		def.Schema = refDef.Schema
		return nil
	}
	for key, schema := range def.Schema.Properties {
		if len(schema.Ref.String()) == 0 {
			continue
		}
		refDef, ok := definitions[def.Schema.Ref.String()]
		if !ok {
			return fmt.Errorf("dependency %q for property %q cannot be found in definitions", def.Schema.Ref.String(), key)
		}
		if err := inlineDependencies(&refDef, definitions); err != nil {
			return fmt.Errorf("dependency %q cannot be inlined: %w", def.Schema.Ref.String(), err)
		}
		def.Schema.Properties[key] = refDef.Schema
	}
	return nil
}

