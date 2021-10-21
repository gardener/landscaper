// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema_test

import (
	"bytes"
	"compress/gzip"
	"os"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/apis/mediatype"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"

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

		It("should pass with a blueprint reference within the schema", func() {
			localSchema1 := []byte(`{ "$ref": "blueprint://myfile2" }`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile1", localSchema1, os.ModePerm)).To(Succeed())

			localSchema2 := []byte(`{ "type": "string"}`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile2", localSchema2, os.ModePerm)).To(Succeed())

			schemaBytes := []byte(`{ "$ref": "blueprint://myfile1" }`)
			data := []byte(`"abc"`)

			Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
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

		It("should resolve a local schema in a local schema", func() {
			config = &jsonschema.LoaderConfig{
				LocalTypes: map[string]lsv1alpha1.JSONSchemaDefinition{
					"mycustom": {RawMessage: []byte(`{ "type": "string"}`)},
					"indirect": {RawMessage: []byte(`{ "$ref": "local://mycustom"}`)},
				},
			}
			validator = &jsonschema.Validator{
				Config: config,
			}
			schemaBytes := []byte(`{ "$ref": "local://indirect"}`)
			data := []byte(`"valid"`)

			Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
		})

		It("should resolve a local schema in a complex local schema", func() {
			config = &jsonschema.LoaderConfig{
				LocalTypes: map[string]lsv1alpha1.JSONSchemaDefinition{
					"resourceRequirements": {RawMessage: []byte(`
{
    "type": "object",
    "properties": {
      "limits": {
        "$ref": "local://resourceList"
      },
      "requests": {
        "$ref": "local://resourceList"
      }
    }
  }
`)},
					"resourceList": {RawMessage: []byte(`
{
    "type": "object",
    "properties": {
      "cpu": {
        "type": "string"
      },
      "memory": {
        "type": "string"
      }
    }
  }
`)},
				},
			}
			validator = &jsonschema.Validator{
				Config: config,
			}
			schemaBytes := []byte(`{ "$ref": "local://resourceRequirements"}`)
			data := []byte(`
{
  "limits": {
    "cpu": "100m",
    "memory": "1G"
  }
}
`)

			Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
		})

		It("should resolve a local schema in a complex local schema and correctly validate a invalid schema", func() {
			config = &jsonschema.LoaderConfig{
				LocalTypes: map[string]lsv1alpha1.JSONSchemaDefinition{
					"resourceRequirements": {RawMessage: []byte(`
{
    "type": "object",
    "properties": {
      "limits": {
        "$ref": "local://resourceList"
      },
      "requests": {
        "$ref": "local://resourceList"
      }
    }
  }
`)},
					"resourceList": {RawMessage: []byte(`
{
    "type": "object",
    "properties": {
      "cpu": {
        "type": "string"
      },
      "memory": {
        "type": "string"
      }
    }
  }
`)},
				},
			}
			validator = &jsonschema.Validator{
				Config: config,
			}
			schemaBytes := []byte(`{ "$ref": "local://resourceRequirements"}`)
			data := []byte(`
{
  "limits": {
    "cpu": 1,
    "memory": "1G"
  }
}
`)
			err := validator.ValidateBytes(schemaBytes, data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("limits.cpu"))
			Expect(err.Error()).To(ContainSubstring("Invalid value: 1: Invalid type. Expected: string, given: integer"))
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

	Context("ComponentDescriptorReference", func() {
		var (
			config *jsonschema.LoaderConfig
			blobFs vfs.FileSystem
		)
		BeforeEach(func() {
			blobFs = memoryfs.New()
			Expect(blobFs.Mkdir(ctf.BlobsDirectoryName, os.ModePerm)).To(Succeed())

			repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/reg", ""))
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/test"
			cd.Version = "v0.0.0"
			cd.RepositoryContexts = []*cdv2.UnstructuredTypedObject{&repoCtx}
			compRes, err := ctf.NewListResolver(&cdv2.ComponentDescriptorList{
				Components: []cdv2.ComponentDescriptor{*cd},
			}, componentsregistry.NewLocalFilesystemBlobResolver(blobFs))
			Expect(err).To(Not(HaveOccurred()))

			localSchema := []byte(`{ "type": "string"}`)
			Expect(vfs.WriteFile(blobFs, ctf.BlobPath("default.json"), localSchema, os.ModePerm)).To(Succeed())
			acc, err := cdv2.NewUnstructured(cdv2.NewLocalFilesystemBlobAccess("default.json", mediatype.JSONSchemaArtifactsMediaTypeV1))
			Expect(err).ToNot(HaveOccurred())
			cd.Resources = append(cd.Resources, cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "default",
					Version: cd.Version,
				},
				Relation: cdv2.LocalRelation,
				Access:   &acc,
			})

			config = &jsonschema.LoaderConfig{
				ComponentDescriptor: cd,
				ComponentResolver:   compRes,
			}

			validator = &jsonschema.Validator{
				Config: config,
			}
		})

		It("should pass with a schema from a component descriptor resource", func() {
			schemaBytes := []byte(`{ "$ref": "cd://resources/default"}`)
			data := []byte(`"valid"`)

			Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
		})

		It("should fail with a schema from a blueprint file reference", func() {
			schemaBytes := []byte(`{ "$ref": "cd://resources/default"}`)
			data := []byte("7")

			Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
		})

		It("should fail when the configured blueprint file reference cannot be found", func() {
			schemaBytes := []byte(`{ "$ref": "cd://resources/fail"}`)
			data := []byte("7")

			Expect(validator.ValidateBytes(schemaBytes, data)).To(HaveOccurred())
		})

		It("should pass with a gzip compressed schema from a component descriptor resource", func() {
			var localSchema bytes.Buffer

			w := gzip.NewWriter(&localSchema)
			_, err := w.Write([]byte(`{ "type": "string"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Close()).To(Succeed())

			Expect(vfs.WriteFile(blobFs, ctf.BlobPath("default.json"), localSchema.Bytes(), os.ModePerm)).To(Succeed())
			acc, err := cdv2.NewUnstructured(cdv2.NewLocalFilesystemBlobAccess("default.json",
				mediatype.NewBuilder(mediatype.JSONSchemaArtifactsMediaTypeV1).Compression(mediatype.GZipCompression).Build().String()))
			Expect(err).ToNot(HaveOccurred())
			config.ComponentDescriptor.Resources = append(config.ComponentDescriptor.Resources, cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "comp",
					Version: config.ComponentDescriptor.Version,
				},
				Relation: cdv2.LocalRelation,
				Access:   &acc,
			})

			schemaBytes := []byte(`{ "$ref": "cd://resources/comp"}`)
			data := []byte(`"valid"`)

			Expect(validator.ValidateBytes(schemaBytes, data)).To(Succeed())
		})

		It("should throw an error if a wrong media type is used", func() {
			Expect(vfs.WriteFile(blobFs, ctf.BlobPath("default.json"), []byte(`{ "type": "string"}`), os.ModePerm)).To(Succeed())
			acc, err := cdv2.NewUnstructured(cdv2.NewLocalFilesystemBlobAccess("default.json", "application/unknown"))
			Expect(err).ToNot(HaveOccurred())
			config.ComponentDescriptor.Resources = append(config.ComponentDescriptor.Resources, cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "unknown",
					Version: config.ComponentDescriptor.Version,
				},
				Relation: cdv2.LocalRelation,
				Access:   &acc,
			})

			schemaBytes := []byte(`{ "$ref": "cd://resources/unknown"}`)
			data := []byte(`"valid"`)

			err = validator.ValidateBytes(schemaBytes, data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown media type"))
		})

	})

})
