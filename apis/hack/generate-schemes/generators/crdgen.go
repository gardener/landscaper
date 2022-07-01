// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generators

import (
	"context"
	"encoding/json"
	"fmt"

	lsschema "github.com/gardener/landscaper/apis/schema"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextval "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"k8s.io/utils/pointer"
)

// CustomResourceDefinition defines the internal representation of the custom resource.
type CustomResourceDefinition struct {
	OutputDir string
	CRD       *apiextv1.CustomResourceDefinition
}

// CRDGenerator defines a generator to generate crd's.
type CRDGenerator struct {
	crds        map[string]*CustomResourceDefinition
	Definitions map[string]common.OpenAPIDefinition
}

// NewCRDGenerator creates a new crd generator.
func NewCRDGenerator(definitionsFunc func(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition) *CRDGenerator {
	origRefCallback := func(path string) spec.Ref {
		return spec.MustCreateRef(path)
	}
	return &CRDGenerator{
		crds:        map[string]*CustomResourceDefinition{},
		Definitions: definitionsFunc(origRefCallback),
	}
}

func (g *CRDGenerator) Generate(group, version string, def lsschema.CustomResourceDefinition, outputDir string) error {

	// first find the correct schema definition
	var (
		schema *spec.Schema
		pvn    PackageVersionName
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

	// remove metadata and apiversion fields
	if _, ok := schema.Properties["metadata"]; ok {
		delete(schema.Properties, "metadata")
	}
	if _, ok := schema.Properties["apiVersion"]; ok {
		delete(schema.Properties, "apiVersion")
	}
	if _, ok := schema.Properties["kind"]; ok {
		delete(schema.Properties, "kind")
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
		crd = &CustomResourceDefinition{
			OutputDir: outputDir,
			CRD:       &apiextv1.CustomResourceDefinition{},
		}
		crd.CRD.APIVersion = "apiextensions.k8s.io/v1"
		crd.CRD.Kind = "CustomResourceDefinition"
		crd.CRD.Name = crdName
		crd.CRD.Spec.Group = group
		crd.CRD.Spec.Scope = apiextv1.ResourceScope(def.Scope)
		crd.CRD.Spec.Names = apiextv1.CustomResourceDefinitionNames{
			Plural:     def.Names.Plural,
			Singular:   def.Names.Singular,
			ShortNames: def.Names.ShortNames,
			Kind:       def.Names.Kind,
			ListKind:   def.Names.ListKind,
			Categories: def.Names.Categories,
		}
		crd.CRD.Status.StoredVersions = []string{}
		crd.CRD.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{}
		g.crds[crdName] = crd
	}
	if len(outputDir) != 0 && crd.OutputDir != outputDir {
		return fmt.Errorf("different output directories defined for the same resource %s/%s/%s: %s vs. %s",
			group, version, def.Names.Kind, outputDir, crd.OutputDir)
	}

	defVersion := apiextv1.CustomResourceDefinitionVersion{
		Name:               version,
		Served:             def.Served,
		Storage:            def.Storage,
		Deprecated:         def.Deprecated,
		DeprecationWarning: nil,
		Schema: &apiextv1.CustomResourceValidation{
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
	crd.CRD.Spec.Versions = append(crd.CRD.Spec.Versions, defVersion)
	return nil
}

func (g *CRDGenerator) CRDs() ([]*CustomResourceDefinition, error) {
	crds := make([]*CustomResourceDefinition, 0)
	ctx := context.Background()
	for _, c := range g.crds {
		defaultedCRD := c.CRD.DeepCopy()
		apiextv1.SetDefaults_CustomResourceDefinition(defaultedCRD)
		coreCRD := &apiext.CustomResourceDefinition{}
		if err := apiextv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(defaultedCRD, coreCRD, nil); err != nil {
			return nil, fmt.Errorf("unable to convert crd %s: %w", c.CRD.Name, err)
		}
		if err := apiextval.ValidateCustomResourceDefinition(ctx, coreCRD); len(err) != 0 {
			return nil, fmt.Errorf("crd %s is invalid: %w", c.CRD.Name, err.ToAggregate())
		}
		crds = append(crds, &CustomResourceDefinition{
			OutputDir: c.OutputDir,
			CRD:       c.CRD,
		})
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
			Description:            schema.Description,
			XPreserveUnknownFields: pointer.BoolPtr(true),
		}, nil
	}

	// automatically use runtime extension if the schema is one
	if schema.Type[0] == "object" && schema.Ref.String() == "k8s.io/apimachinery/pkg/runtime.RawExtension" {
		return &apiextv1.JSONSchemaProps{
			Type:                   "object",
			Description:            schema.Description,
			XPreserveUnknownFields: pointer.BoolPtr(true),
			XEmbeddedResource:      true,
		}, nil
	}
	if schema.Ref.String() == "github.com/gardener/component-spec/bindings-go/apis/v2.UnstructuredTypedObject" {
		return &apiextv1.JSONSchemaProps{
			Type:                   "object",
			Description:            schema.Description,
			XPreserveUnknownFields: pointer.BoolPtr(true),
		}, nil
	}
	schemaProps := &apiextv1.JSONSchemaProps{
		ID:                   schema.ID,
		Schema:               "",
		Ref:                  nil, // never set in our case
		Description:          schema.Description,
		Type:                 schema.Type[0],
		Format:               schema.Format,
		Title:                "",
		Default:              nil,
		Maximum:              schema.Maximum,
		ExclusiveMaximum:     schema.ExclusiveMaximum,
		Minimum:              schema.Minimum,
		ExclusiveMinimum:     schema.ExclusiveMinimum,
		MaxLength:            schema.MaxLength,
		MinLength:            schema.MinLength,
		Pattern:              schema.Pattern,
		MaxItems:             schema.MaxItems,
		MinItems:             schema.MinItems,
		UniqueItems:          schema.UniqueItems,
		MultipleOf:           schema.MultipleOf,
		MaxProperties:        schema.MaxProperties,
		MinProperties:        schema.MinProperties,
		Required:             schema.Required,
		Properties:           nil,
		AdditionalProperties: nil,
		PatternProperties:    nil,
		AdditionalItems:      nil,
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
