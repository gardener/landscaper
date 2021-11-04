package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gardener/landscaper/apis/hack/generate-schemes/generators"
	lsschema "github.com/gardener/landscaper/apis/schema"
	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kube-openapi/pkg/common"
	"sigs.k8s.io/yaml"
)

type (
	// GetOpenAPIDefinitions is the open api callback to retrieve the definitions
	GetOpenAPIDefinitions = func(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition
)

// SchemaGenerator generates JSON schemas and Custom Resource Definitions
type SchemaGenerator struct {
	// Exports defines the types for which the schemas shall be generated
	Exports []string

	// CRDs specifies the custom resource definition which is used for generating the schemas
	CRDs []lsschema.CustomResourceDefinitions

	// GetOpenAPIDefinitions is a callback to the OpenAPI definitions
	GetOpenAPIDefinitions GetOpenAPIDefinitions
}

// NewSchemaGenerator creates a new SchemaGenerator instance
func NewSchemaGenerator(exports []string, CRDs []lsschema.CustomResourceDefinitions, getOpenAPIDefinitions GetOpenAPIDefinitions) *SchemaGenerator {
	schemaGenerator := &SchemaGenerator{
		Exports: exports,
		CRDs:    CRDs,
		GetOpenAPIDefinitions: getOpenAPIDefinitions,
	}
	return schemaGenerator
}

// Run runs the schema generator with the given output directories
func (g *SchemaGenerator) Run(schemaDir, crdDir string) error {
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
		Definitions: g.GetOpenAPIDefinitions(refCallback),
	}
	for defName, apiDefinition := range jsonGen.Definitions {
		if !ShouldCreateDefinition(g.Exports, defName) {
			continue
		}
		data, err := jsonGen.Generate(defName, apiDefinition)
		if err != nil {
			return fmt.Errorf("unable to generate jsonschema for %s: %w", defName, err)
		}

		// write file
		file := filepath.Join(schemaDir, generators.ParsePackageVersionName(defName).String()+".json")
		if err := ioutil.WriteFile(file, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write jsonschema for %q to %q: %w", file, generators.ParsePackageVersionName(defName).String(), err)
		}

		fmt.Printf("Generated jsonschema for %q...\n", generators.ParsePackageVersionName(defName).String())
	}

	if len(crdDir) == 0 {
		log.Println("Skip crd generation")
		return nil
	}
	crdGen := generators.NewCRDGenerator(g.GetOpenAPIDefinitions)
	for _, crdVersion := range g.CRDs {
		for _, crdDef := range crdVersion.Definitions {
			if err := crdGen.Generate(crdVersion.Group, crdVersion.Version, crdDef, crdVersion.OutputDir); err != nil {
				return fmt.Errorf("unable to generate crd for %s %s %s: %w", crdVersion.Group, crdVersion.Version, crdDef.Names.Kind, err)
			}
		}
	}

	crds, err := crdGen.CRDs()
	if err != nil {
		return err
	}
	cleanedCrdDirs := sets.NewString(crdDir)
	for _, crd := range crds {
		jsonBytes, err := json.Marshal(crd.CRD)
		if err != nil {
			return fmt.Errorf("unable to marshal CRD %s: %w", crd.CRD.Name, err)
		}
		data, err := yaml.JSONToYAML(jsonBytes)
		if err != nil {
			return fmt.Errorf("unable to convert json of CRD %s to yaml: %w", crd.CRD.Name, err)
		}

		outDir := crdDir
		if len(crd.OutputDir) != 0 {
			outDir = crd.OutputDir
		}
		if !cleanedCrdDirs.Has(outDir) {
			if err := prepareExportDir(outDir); err != nil {
				return err
			}
			cleanedCrdDirs.Insert(outDir)
		}

		// write file
		file := filepath.Join(outDir, fmt.Sprintf("%s_%s.yaml", crd.CRD.Spec.Group, crd.CRD.Spec.Names.Plural))
		if err := ioutil.WriteFile(file, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write crd for %q to %q: %w", file, crd.CRD.Name, err)
		}

		fmt.Printf("Generated crd for %q in %s...\n", crd.CRD.Name, file)
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
	log.Printf("Prepate export dir %q", exportDir)
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
