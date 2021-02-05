// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema_test

import (
	"os"
	"testing"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
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

	Context("BlueprintReferenceTemplate", func() {
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
					"mycustom": {RawMessage: []byte(`{ "type": "string"}`)},
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
					"mycustom": {RawMessage: []byte(`{ "definitions": { "myschema": { "type": "string" } } }`)},
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
