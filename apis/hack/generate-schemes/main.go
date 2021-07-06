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
	lsschema "github.com/gardener/landscaper/apis/schema"
	"github.com/go-openapi/spec"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextval "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/utils/pointer"
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
		ref, _ := jsonreference.New(definitionsRef(path))
		return spec.Ref{Ref: ref}
	}
	jsonGen := &JSONSchemaGenerator{
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
		file := filepath.Join(schemaDir, ParsePackageVersionName(defName).String() + ".json")
		if err := ioutil.WriteFile(file, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write jsonschema for %q to %q: %w", file, ParsePackageVersionName(defName).String(), err)
		}

		fmt.Printf("Generated jsonschema for %q...\n", ParsePackageVersionName(defName).String())
	}

	if len(crdDir) == 0 {
		log.Println("Skip crd generation")
		return nil
	}
	crdGen := NewCRDGenerator(openapi.GetOpenAPIDefinitions)
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

type PackageVersionName struct {
	Name string
	Version string
	Package string
}

// ParsePackageVersionName parses the name from the openapi definition.
// name is expected to be some/path/<package>/<version>.<name>
func ParsePackageVersionName(name string) PackageVersionName {
	splitName := strings.Split(name, "/")
	if len(splitName) < 2 {
		panic(fmt.Errorf("a component name must consits of at least a package identifier and a name.version but got %s", name))
	}
	versionName := splitName[len(splitName) - 1]
	packageName := splitName[len(splitName) - 2]

	versionNameSplit := strings.Split(versionName, ".")
	if len(versionNameSplit) != 2 {
		panic(fmt.Errorf("a component name must consits of name.version but got %s", versionName))
	}
	return PackageVersionName{
		Package: packageName,
		Name: versionNameSplit[1],
		Version: versionNameSplit[0],
	}
}

// String implements the stringer method.
func (pvn PackageVersionName) String() string {
	return fmt.Sprintf("%s-%s-%s", pvn.Package, pvn.Version, pvn.Name)
}

// definitionsRef returns the reference to the resource of a specific object name
func definitionsRef(name string) string {
	pvn := ParsePackageVersionName(name).String()
	return fmt.Sprintf("#/%s/%s", DefinitionsPropName, pvn)
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

// CRDGenerator defines a generator to generate crd's.
type CRDGenerator struct {
	crds map[string]*apiextv1.CustomResourceDefinition
	Definitions map[string]common.OpenAPIDefinition
}

// NewCRDGenerator creates a new crd generator.
func NewCRDGenerator(definitionsFunc func(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition) *CRDGenerator {
	origRefCallback := func(path string) spec.Ref {
		return spec.MustCreateRef(path)
	}
	return &CRDGenerator{
		crds: map[string]*apiextv1.CustomResourceDefinition{},
		Definitions: definitionsFunc(origRefCallback),
	}
}

func (g *CRDGenerator) Generate(group, version string, def lsschema.CustomResourceDefinition) error {

	// first find the correct schema definition
	var (
		schema *spec.Schema
		pvn PackageVersionName
	)
	for name, openapidef := range g.Definitions {
		pvn = ParsePackageVersionName(name)
		if pvn.Name == def.Names.Kind && pvn.Version == version {
			schema = &openapidef.Schema
			break
		}
	}
	if schema == nil {
		return fmt.Errorf("no openapi schema found for %s %s %s", group, version, def.Names.Kind)
	}

	// remove metadata field
	if _, ok := schema.Properties["metadata"]; ok {
		delete(schema.Properties, "metadata")
	}
	if err := g.resolveSchema(schema); err != nil {
		return fmt.Errorf("unable to add dependencies: %w", err)
	}

	jsonSchemaProps, err := ConvertSpecSchemaToApiextv1Schema(schema)
	if err != nil {
		return fmt.Errorf("unable to convert jsonschema of %s %s %s: %w", group, version, def.Names.Kind, err)
	}

	crdName := fmt.Sprintf("%s.%s", def.Names.Plural, group)
	crd, ok := g.crds[crdName]
	if !ok {
		crd = &apiextv1.CustomResourceDefinition{}
		crd.APIVersion = "apiextensions.k8s.io/v1"
		crd.Kind = "CustomResourceDefinition"
		crd.Name = crdName
		crd.Spec.Group = group
		crd.Spec.Scope = apiextv1.ResourceScope(def.Scope)
		crd.Spec.Names = apiextv1.CustomResourceDefinitionNames{
			Plural:     def.Names.Plural,
			Singular:   def.Names.Singular,
			ShortNames: def.Names.ShortNames,
			Kind:       def.Names.Kind,
			ListKind:   def.Names.ListKind,
			Categories: def.Names.Categories,
		}
		g.crds[crdName] = crd
	}
	defVersion := apiextv1.CustomResourceDefinitionVersion{
		Name:                     version,
		Served:                   def.Served,
		Storage:                  def.Storage,
		Deprecated:               def.Deprecated,
		DeprecationWarning:       nil,
		Schema:                   &apiextv1.CustomResourceValidation{
			OpenAPIV3Schema: jsonSchemaProps,
		},
	}
	if def.SubresourceStatus {
		defVersion.Subresources = &apiextv1.CustomResourceSubresources{
			Status: &apiextv1.CustomResourceSubresourceStatus{},
		}
	}
	for _, col := range def.AdditionalPrinterColumns {
		defVersion.AdditionalPrinterColumns = append(defVersion.AdditionalPrinterColumns, apiextv1.CustomResourceColumnDefinition{
			Name:        col.Name,
			Type:        col.Type,
			Format:      col.Format,
			Description: col.Description,
			Priority:    col.Priority,
			JSONPath:    col.JSONPath,
		})
	}
	crd.Spec.Versions = append(crd.Spec.Versions, defVersion)
	return nil
}

func (g *CRDGenerator) CRDs() ([]*apiextv1.CustomResourceDefinition, error) {
	crds := make([]*apiextv1.CustomResourceDefinition, 0)
	for _, c := range g.crds {
		defaultedCRD := c.DeepCopy()
		apiextv1.SetDefaults_CustomResourceDefinition(defaultedCRD)
		coreCRD := &apiext.CustomResourceDefinition{}
		if err := apiextv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(defaultedCRD, coreCRD, nil); err != nil {
			return nil, fmt.Errorf("unable to convert crd %s: %w", c.Name, err)
		}
		if err := apiextval.ValidateCustomResourceDefinition(coreCRD, runtimeschema.GroupVersion{
			Group:   c.GroupVersionKind().Group,
			Version: c.GroupVersionKind().Version,
		}); len(err) != 0 {
			return nil, fmt.Errorf("crd %s is invalid: %w", c.Name, err.ToAggregate())
		}
		crds = append(crds, c)
	}
	return crds, nil
}

// inlineDependencies adds all dependencies of the given definition to the map of schemas
func (g *CRDGenerator) resolveSchema(schema *spec.Schema) error {
	if schema == nil {
		return nil
	}
	resolve := func(schema *spec.Schema) error {
		if schema == nil {
			return nil
		}
		ref := schema.Ref
		refStr := ref.String()
		if len(refStr) == 0 {
			return nil
		}
		refDef, ok := g.Definitions[refStr]
		if !ok {
			return fmt.Errorf("dependency %q cannot be found in definitions", refStr)
		}
		if err := g.resolveSchema(&refDef.Schema); err != nil {
			return fmt.Errorf("dependency %q cannot be inlined: %w", refStr, err)
		}
		if len(schema.Description) != 0 {
			refDef.Schema.Description = schema.Description
		}
		*schema = refDef.Schema
		schema.Ref = ref
		return nil
	}
	if len(schema.Ref.String()) != 0 {
		// resolve complete schema
		return resolve(schema)
	}

	// alternatively check if the schema is defined via items
	if schema.Items != nil && schema.Items.Schema != nil {
		if err := g.resolveSchema(schema.Items.Schema); err != nil {
			return err
		}
	}
	for key := range schema.Properties {
		refSchema := schema.Properties[key]
		if err := g.resolveSchema(&refSchema); err != nil {
			return err
		}
		schema.Properties[key] = refSchema
	}
	if schema.AdditionalProperties != nil {
		if err := g.resolveSchema(schema.AdditionalProperties.Schema); err != nil {
			return err
		}
	}
	if schema.AdditionalItems != nil {
		if err := g.resolveSchema(schema.AdditionalItems.Schema); err != nil {
			return err
		}
	}

	// replace array
	if schema.Items == nil {
		return nil
	}
	if schema.Items.Schema != nil {
		if err := g.resolveSchema(schema.Items.Schema); err != nil {
			return err
		}
	}
	for i := range schema.Items.Schemas {
		refSchema := schema.Items.Schemas[i]
		if err := g.resolveSchema(&refSchema); err != nil {
			return err
		}
		schema.Items.Schemas[i] = refSchema
	}
	return nil
}

func ConvertSpecSchemaToApiextv1Schema(schema *spec.Schema) (*apiextv1.JSONSchemaProps, error) {
	if schema == nil {
		return nil, nil
	}
	if len(schema.Type) > 1 {
		return &apiextv1.JSONSchemaProps{
			Description: schema.Description,
			XPreserveUnknownFields: pointer.BoolPtr(true),
		}, nil
	}

	// automatically use runtime extension if the schema is one
	if schema.Type[0] == "object" && schema.Ref.String() == "k8s.io/apimachinery/pkg/runtime.RawExtension" {
		return &apiextv1.JSONSchemaProps{
			Type: "object",
			Description: schema.Description,
			XPreserveUnknownFields: pointer.BoolPtr(true),
			XEmbeddedResource: true,
		}, nil
	}
	schemaProps := &apiextv1.JSONSchemaProps{
		ID:                     schema.ID,
		Schema:                 "",
		Ref:                    nil, // never set in our case
		Description:            schema.Description,
		Type:                   schema.Type[0],
		Format:                 schema.Format,
		Title:                  "",
		Default:                nil,
		Maximum:                schema.Maximum,
		ExclusiveMaximum:       schema.ExclusiveMaximum,
		Minimum:                schema.Minimum,
		ExclusiveMinimum:       schema.ExclusiveMinimum,
		MaxLength:              schema.MaxLength,
		MinLength:              schema.MinLength,
		Pattern:               schema.Pattern,
		MaxItems:               schema.MaxItems,
		MinItems:               schema.MinItems,
		UniqueItems:            schema.UniqueItems,
		MultipleOf:             schema.MultipleOf,
		MaxProperties:          schema.MaxProperties,
		MinProperties:          schema.MinProperties,
		Required:               schema.Required,
		Properties:             nil,
		AdditionalProperties:   nil,
		PatternProperties:      nil,
		AdditionalItems:        nil,
	}

	if schema.Enum != nil {
		for i, enum := range schema.Enum {
			enumBytes, err := json.Marshal(enum)
			if err != nil {
				return nil, fmt.Errorf("unable to marshal enum %d of schema: %w", i, err)
			}
			schemaProps.Enum = append(schemaProps.Enum, apiextv1.JSON{Raw: enumBytes})
		}
	}

	if schema.Items != nil {
		if schema.Items.Schema != nil {
			cItem, err := ConvertSpecSchemaToApiextv1Schema(schema.Items.Schema)
			if err != nil {
				return nil, fmt.Errorf("unable to convert item: %w", err)
			}
			schemaProps.Items = &apiextv1.JSONSchemaPropsOrArray{
				Schema: cItem,
			}
		} else {
			schemaProps.Items = &apiextv1.JSONSchemaPropsOrArray{}
			for i, item := range schema.Items.Schemas {
				cItem, err := ConvertSpecSchemaToApiextv1Schema(&item)
				if err != nil {
					return nil, fmt.Errorf("unable to convert item %d: %w", i, err)
				}
				schemaProps.Items.JSONSchemas = append(schemaProps.Items.JSONSchemas, *cItem)
			}
		}
	}

	if schema.Properties != nil {
		schemaProps.Properties = map[string]apiextv1.JSONSchemaProps{}
		for key, item := range schema.Properties {
			cItem, err := ConvertSpecSchemaToApiextv1Schema(&item)
			if err != nil {
				return nil, fmt.Errorf("unable to convert property item %s: %w", key, err)
			}
			schemaProps.Properties[key] = *cItem
		}
	}

	for i, item := range schema.AllOf {
		cItem, err := ConvertSpecSchemaToApiextv1Schema(&item)
		if err != nil {
			return nil, fmt.Errorf("unable to convert allOf item %d: %w", i, err)
		}
		schemaProps.AllOf = append(schemaProps.AllOf, *cItem)
	}
	for i, item := range schema.OneOf {
		cItem, err := ConvertSpecSchemaToApiextv1Schema(&item)
		if err != nil {
			return nil, fmt.Errorf("unable to convert oneOf item %d: %w", i, err)
		}
		schemaProps.AllOf = append(schemaProps.OneOf, *cItem)
	}
	for i, item := range schema.AnyOf {
		cItem, err := ConvertSpecSchemaToApiextv1Schema(&item)
		if err != nil {
			return nil, fmt.Errorf("unable to convert anyOf item %d: %w", i, err)
		}
		schemaProps.AllOf = append(schemaProps.AnyOf, *cItem)
	}

	if schema.Not != nil {
		s, err := ConvertSpecSchemaToApiextv1Schema(schema.Not)
		if err != nil {
			return nil, fmt.Errorf("unable to convert Not schema: %w", err)
		}
		schemaProps.Not = s
	}

	if schema.AdditionalProperties != nil {
		s, err := ConvertSpecSchemaToApiextv1Schema(schema.AdditionalProperties.Schema)
		if err != nil {
			return nil, fmt.Errorf("unable to convert additonal properties schema: %w", err)
		}
		schemaProps.AdditionalProperties = &apiextv1.JSONSchemaPropsOrBool{
			Allows: schema.AdditionalProperties.Allows,
			Schema: s,
		}
	}
	if schema.AdditionalItems != nil {
		s, err := ConvertSpecSchemaToApiextv1Schema(schema.AdditionalItems.Schema)
		if err != nil {
			return nil, fmt.Errorf("unable to convert additonal items schema: %w", err)
		}
		schemaProps.AdditionalItems = &apiextv1.JSONSchemaPropsOrBool{
			Allows: schema.AdditionalItems.Allows,
			Schema: s,
		}
	}

	return schemaProps, nil
}

