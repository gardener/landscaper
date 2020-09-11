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

package jsonschema_test

import (
	"os"
	"testing"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Test Suite")
}

var _ = Describe("jsonschema", func() {

	var validator *jsonschema.Validator

	BeforeEach(func() {
		validator = &jsonschema.Validator{Config: &jsonschema.LoaderConfig{}}
	})

	Context("schema validation", func() {
		It("should pass a correct schema", func() {
			schemaBytes := []byte(`{ "type": "string"}`)
			Expect(jsonschema.ValidateSchema(schemaBytes)).To(Succeed())
		})
		It("should forbid a invalid schema", func() {
			schemaBytes := []byte(`{ "type": 7}`)
			Expect(jsonschema.ValidateSchema(schemaBytes)).To(HaveOccurred())
		})
	})

	It("should pass a simple string", func() {
		schemaBytes := []byte(`{ "type": "string"}`)
		data := []byte(`"string"`)

		Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
	})

	It("should pass a simple number", func() {
		schemaBytes := []byte(`{ "type": "number"}`)
		data := []byte("7")

		Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
	})

	It("should forbid a number as string", func() {
		schemaBytes := []byte(`{ "type": "string"}`)
		data := []byte("7")

		Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
	})

	Context("BlueprintReference", func() {
		var config *jsonschema.LoaderConfig
		BeforeEach(func() {
			config = &jsonschema.LoaderConfig{
				BlueprintFs: memoryfs.New(),
			}
			validator = &jsonschema.Validator{
				Config: config,
			}
		})
		It("should pass with a schema from a blueprint file reference", func() {
			localSchema := []byte(`{ "type": "string"}`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile", localSchema, os.ModePerm)).To(Succeed())

			schemaBytes := []byte(`{ "$ref": "blueprint://myfile"}`)
			data := []byte(`"valid"`)

			Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
		})

		It("should fail with a schema from a blueprint file reference", func() {
			localSchema := []byte(`{ "type": "string"}`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile", localSchema, os.ModePerm)).To(Succeed())

			schemaBytes := []byte(`{ "$ref": "blueprint://myfile"}`)
			data := []byte("7")

			Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
		})

		It("should fail when the configured blueprint file reference cannot be found", func() {
			schemaBytes := []byte(`{ "$ref": "blueprint://myfile"}`)
			data := []byte("7")

			Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
		})

		It("should pass with a local definition reference", func() {
			schemaBytes := []byte(`{ "definitions": { "myschema": { "type": "string" } }, "$ref": "#/definitions/myschema"}`)
			data := []byte("7")

			Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
		})

		It("should validate with a definition local reference in a blueprint file reference", func() {
			localSchema := []byte(`{ "definitions": { "myschema": { "type": "string" } } }`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile", localSchema, os.ModePerm)).To(Succeed())
			schemaBytes := []byte(`{"$ref": "blueprint://myfile#/definitions/myschema"}`)
			pass := []byte(`"abc"`)
			fail := []byte(`7`)

			Expect(validator.ValidateBytes(schemaBytes, pass)).To(Succeed())
			Expect(validator.ValidateBytes(schemaBytes, fail)).To(HaveOccurred())
		})
	})

	Context("LocalReference", func() {
		var config *jsonschema.LoaderConfig
		BeforeEach(func() {
			config = &jsonschema.LoaderConfig{
				LocalTypes: map[string]lsv1alpha1.JSONSchemaDefinition{
					"mycustom": []byte(`{ "type": "string"}`),
				},
			}
			validator = &jsonschema.Validator{
				Config: config,
			}
		})
		It("should pass with a schema from a local reference", func() {
			schemaBytes := []byte(`{ "$ref": "local://mycustom"}`)
			data := []byte(`"valid"`)

			Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
		})

		It("should fail with a schema from a blueprint file reference", func() {
			schemaBytes := []byte(`{ "$ref": "local://mycustom"}`)
			data := []byte("7")

			Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
		})

		It("should fail when the configured blueprint file reference cannot be found", func() {
			schemaBytes := []byte(`{ "$ref": "local://fail"}`)
			data := []byte("7")

			Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
		})

		It("should pass with a local definition reference", func() {
			schemaBytes := []byte(`{ "definitions": { "myschema": { "type": "string" } }, "$ref": "local://mycustom"}}`)
			data := []byte("7")

			Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
		})

		It("should validate a definition local reference in a blueprint file reference", func() {
			config = &jsonschema.LoaderConfig{
				LocalTypes: map[string]lsv1alpha1.JSONSchemaDefinition{
					"mycustom": []byte(`{ "definitions": { "myschema": { "type": "string" } } }`),
				},
			}
			validator = &jsonschema.Validator{
				Config: config,
			}
			schemaBytes := []byte(`{"$ref": "local://mycustom#/definitions/myschema"}`)
			pass := []byte(`"abc"`)
			fail := []byte(`7`)

			Expect(validator.ValidateBytes(schemaBytes, pass)).To(Succeed())
			Expect(validator.ValidateBytes(schemaBytes, fail)).To(HaveOccurred())
		})
	})

})
