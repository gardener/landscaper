// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	depv1alpha1 "github.com/gardener/landscaper/apis/deployer/core/v1alpha1"
	"github.com/gardener/landscaper/apis/hack/generate-schemes/app"
	"github.com/gardener/landscaper/apis/openapi"
	lsschema "github.com/gardener/landscaper/apis/schema"
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
	depv1alpha1.ResourceDefinition,
}

var (
	schemaDir string
	crdDir    string
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
	schemaGenerator := app.NewSchemaGenerator(Exports, CRDs, openapi.GetOpenAPIDefinitions)
	if err := schemaGenerator.Run(schemaDir, crdDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
