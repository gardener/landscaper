// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generators

import (
	"encoding/json"
	"fmt"

	"github.com/go-openapi/spec"
	"k8s.io/kube-openapi/pkg/common"
)


// DefinitionsPropName is the name of property where the definition is located
const DefinitionsPropName = "definitions"

// JSONSchemaGenerator is a jsonschema generator using openapi definitions.
type JSONSchemaGenerator struct {
	Definitions map[string]common.OpenAPIDefinition
}

func (g *JSONSchemaGenerator) Generate(objName string, def common.OpenAPIDefinition) ([]byte, error) {
	pvn := ParsePackageVersionName(objName).String()
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
	if err := g.addDependencies(dependencyDefinitions, objName); err != nil {
		return nil, fmt.Errorf("unable to add dependencies: %w", err)
	}
	jsonSchema[DefinitionsPropName] = dependencyDefinitions
	return json.MarshalIndent(jsonSchema, "", "  ")
}

// addDependencies adds all dependencies of the given definition to the map of schemas
func (g *JSONSchemaGenerator) addDependencies(schemas map[string]spec.Schema, defName string) error {
	def, ok := g.Definitions[defName]
	if !ok {
		return fmt.Errorf("definition %q is not defined", defName)
	}
	for _, depName := range def.Dependencies {
		if _, ok :=schemas[ParsePackageVersionName(depName).String()]; ok {
			continue
		}
		def, ok := g.Definitions[depName]
		if !ok {
			return fmt.Errorf("dependency %q of %q cannot be found in definitions", depName, defName)
		}
		schemas[ParsePackageVersionName(depName).String()] = def.Schema
		// add dependencies for that component if not already defined
		if err := g.addDependencies(schemas, depName); err != nil {
			return err
		}
	}
	return nil
}

// DefinitionsRef returns the reference to the resource of a specific object name
func DefinitionsRef(name string) string {
	pvn := ParsePackageVersionName(name).String()
	return fmt.Sprintf("#/%s/%s", DefinitionsPropName, pvn)
}
