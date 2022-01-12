// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/apis/mediatype"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	testcred "github.com/gardener/component-cli/ociclient/credentials"
	testreg "github.com/gardener/component-cli/ociclient/test/envtest"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"

	"github.com/gardener/landscaper/test/utils"
	testutils "github.com/gardener/landscaper/test/utils"
)

const jsonschemaResourceType = "landscaper.gardener.cloud/jsonschema"

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Test Suite")
}

var _ = Describe("jsonschema", func() {

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

		Expect(jsonschema.ValidateBytes(schemaBytes, data, nil)).To(Succeed())
	})

	It("should pass a simple number", func() {
		schemaBytes := []byte(`{ "type": "number"}`)
		data := []byte("7")

		Expect(jsonschema.ValidateBytes(schemaBytes, data, nil)).To(Succeed())
	})

	It("should forbid a number as string", func() {
		schemaBytes := []byte(`{ "type": "string"}`)
		data := []byte("7")

		Expect(jsonschema.ValidateBytes(schemaBytes, data, nil)).To(HaveOccurred())
	})

	Context("BlueprintReferenceTemplate", func() {
		var config *jsonschema.ReferenceContext
		BeforeEach(func() {
			config = &jsonschema.ReferenceContext{
				BlueprintFs: memoryfs.New(),
			}
		})
		It("should pass with a schema from a blueprint file reference", func() {
			localSchema := []byte(`{ "type": "string"}`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile", localSchema, os.ModePerm)).To(Succeed())

			schemaBytes := []byte(`{ "$ref": "blueprint://myfile"}`)
			data := []byte(`"valid"`)

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
		})

		It("should fail with a schema from a blueprint file reference", func() {
			localSchema := []byte(`{ "type": "string"}`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile", localSchema, os.ModePerm)).To(Succeed())

			schemaBytes := []byte(`{ "$ref": "blueprint://myfile"}`)
			data := []byte("7")

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(HaveOccurred())
		})

		It("should fail when the configured blueprint file reference cannot be found", func() {
			schemaBytes := []byte(`{ "$ref": "blueprint://myfile"}`)
			data := []byte("7")

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(HaveOccurred())
		})

		It("should pass with a local definition reference", func() {
			schemaBytes := []byte(`{ "definitions": { "myschema": { "type": "string" } }, "$ref": "#/definitions/myschema"}`)
			data := []byte("7")

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(HaveOccurred())
		})

		It("should pass with a blueprint reference within the schema", func() {
			localSchema1 := []byte(`{ "$ref": "blueprint://myfile2" }`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile1", localSchema1, os.ModePerm)).To(Succeed())

			localSchema2 := []byte(`{ "type": "string"}`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile2", localSchema2, os.ModePerm)).To(Succeed())

			schemaBytes := []byte(`{ "$ref": "blueprint://myfile1" }`)
			data := []byte(`"abc"`)

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
		})

		It("should validate with a definition local reference in a blueprint file reference", func() {
			localSchema := []byte(`{ "definitions": { "myschema": { "type": "string" } } }`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile", localSchema, os.ModePerm)).To(Succeed())
			schemaBytes := []byte(`{"$ref": "blueprint://myfile#/definitions/myschema"}`)
			pass := []byte(`"abc"`)
			fail := []byte(`7`)

			Expect(jsonschema.ValidateBytes(schemaBytes, pass, config)).To(Succeed())
			Expect(jsonschema.ValidateBytes(schemaBytes, fail, config)).To(HaveOccurred())
		})
	})

	Context("LocalReference", func() {
		var config *jsonschema.ReferenceContext
		BeforeEach(func() {
			config = &jsonschema.ReferenceContext{
				LocalTypes: map[string]lsv1alpha1.JSONSchemaDefinition{
					"mycustom": {RawMessage: []byte(`{ "type": "string"}`)},
				},
			}
		})

		It("should pass with a schema from a local reference", func() {
			schemaBytes := []byte(`{ "$ref": "local://mycustom"}`)
			data := []byte(`"valid"`)

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
		})

		It("should resolve a local schema in a local schema", func() {
			config = &jsonschema.ReferenceContext{
				LocalTypes: map[string]lsv1alpha1.JSONSchemaDefinition{
					"mycustom": {RawMessage: []byte(`{ "type": "string"}`)},
					"indirect": {RawMessage: []byte(`{ "$ref": "local://mycustom"}`)},
				},
			}
			schemaBytes := []byte(`{ "$ref": "local://indirect"}`)
			data := []byte(`"valid"`)

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
		})

		It("should resolve a local schema in a complex local schema", func() {
			config = &jsonschema.ReferenceContext{
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
			schemaBytes := []byte(`{ "$ref": "local://resourceRequirements"}`)
			data := []byte(`
{
  "limits": {
    "cpu": "100m",
    "memory": "1G"
  }
}
`)

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
		})

		It("should resolve a local schema in a complex local schema and correctly validate a invalid schema", func() {
			config = &jsonschema.ReferenceContext{
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
			schemaBytes := []byte(`{ "$ref": "local://resourceRequirements"}`)
			data := []byte(`
{
  "limits": {
    "cpu": 1,
    "memory": "1G"
  }
}
`)
			err := jsonschema.ValidateBytes(schemaBytes, data, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("limits.cpu"))
			Expect(err.Error()).To(ContainSubstring("Invalid value: 1: Invalid type. Expected: string, given: integer"))
		})

		It("should fail with a schema from a blueprint file reference", func() {
			schemaBytes := []byte(`{ "$ref": "local://mycustom"}`)
			data := []byte("7")

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(HaveOccurred())
		})

		It("should fail when the configured blueprint file reference cannot be found", func() {
			schemaBytes := []byte(`{ "$ref": "local://fail"}`)
			data := []byte("7")

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(HaveOccurred())
		})

		It("should pass with a local definition reference", func() {
			schemaBytes := []byte(`{ "definitions": { "myschema": { "type": "string" } }, "$ref": "local://mycustom"}}`)
			data := []byte("7")

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(HaveOccurred())
		})

		It("should validate a definition local reference in a blueprint file reference", func() {
			config = &jsonschema.ReferenceContext{
				LocalTypes: map[string]lsv1alpha1.JSONSchemaDefinition{
					"mycustom": {RawMessage: []byte(`{ "definitions": { "myschema": { "type": "string" } } }`)},
				},
			}
			schemaBytes := []byte(`{"$ref": "local://mycustom#/definitions/myschema"}`)
			pass := []byte(`"abc"`)
			fail := []byte(`7`)

			Expect(jsonschema.ValidateBytes(schemaBytes, pass, config)).To(Succeed())
			Expect(jsonschema.ValidateBytes(schemaBytes, fail, config)).To(HaveOccurred())
		})
	})

	Context("ComponentDescriptorReference", func() {
		var (
			config *jsonschema.ReferenceContext
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

			config = &jsonschema.ReferenceContext{
				ComponentDescriptor: cd,
				ComponentResolver:   compRes,
			}
		})

		It("should pass with a schema from a component descriptor resource", func() {
			schemaBytes := []byte(`{ "$ref": "cd://resources/default"}`)
			data := []byte(`"valid"`)

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
		})

		It("should fail with a schema from a component descriptor reference", func() {
			schemaBytes := []byte(`{ "$ref": "cd://resources/default"}`)
			data := []byte("7")

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(HaveOccurred())
		})

		It("should fail when the configured component descriptor reference cannot be found", func() {
			schemaBytes := []byte(`{ "$ref": "cd://resources/fail"}`)
			data := []byte("7")

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(HaveOccurred())
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

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
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

			err = jsonschema.ValidateBytes(schemaBytes, data, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown media type"))
		})

	})

	Context("ReferenceResolver", func() {

		It("should not alter references with unknown schemes", func() {
			schema := []byte(`
				{
					"type": "object",
					"properties": {
						"foo": {
							"$ref": "unknown://a/b/c"
						}
					}
				}
			`)
			rr := jsonschema.NewReferenceResolver(nil)
			resolved, err := rr.Resolve(schema)
			testutils.ExpectNoError(err)
			var orig interface{}
			testutils.ExpectNoError(json.Unmarshal(schema, &orig))
			Expect(resolved).To(BeEquivalentTo(orig))
		})
	})

	Context("WithRealRegistry", func() {
		// tests which require a real (local) registry
		var (
			ctx       context.Context
			testenv   *testreg.Environment
			ociCache  cache.Cache
			ociClient ociclient.Client
		)

		BeforeEach(func() {
			ctx = context.Background()

			// create test registry
			testenv = testreg.New(testreg.Options{
				RegistryBinaryPath: filepath.Join("../../../", "tmp", "test", "registry", "registry"),
				Stdout:             GinkgoWriter,
				Stderr:             GinkgoWriter,
			})
			testutils.ExpectNoError(testenv.Start(context.Background()))
			testutils.ExpectNoError(testenv.WaitForRegistryToBeHealthy())

			keyring := testcred.New()
			testutils.ExpectNoError(keyring.AddAuthConfig(testenv.Addr, testcred.AuthConfig{
				Username: testenv.BasicAuth.Username,
				Password: testenv.BasicAuth.Password,
			}))

			var err error
			ociCache, err = cache.NewCache(logr.Discard())
			testutils.ExpectNoError(err)
			ociClient, err = ociclient.NewClient(logr.Discard(), ociclient.WithKeyring(keyring), ociclient.WithCache(ociCache))
			testutils.ExpectNoError(err)
		}, 60)

		AfterEach(func() {
			testutils.ExpectNoError(testenv.Close())
			ctx.Done()
		}, 60)

		It("should correctly resolve nested references across components", func() {
			// create components
			blobPath := "blobs"
			baseConfig := &componentConfig{
				ComponentNameInRegistry:  "example.com/base",
				ComponentNameInReference: "base",
				Version:                  "v0.0.1",
				ReferencedResourceName:   "jsonschemaref",
			}
			firstRefConfig := &componentConfig{
				ComponentNameInRegistry:  "example.com/firstref",
				ComponentNameInReference: "firstref",
				Version:                  "v0.0.2",
				ReferencedResourceName:   "firstjsonschemarefplain",
			}
			secondRefConfig := &componentConfig{
				ComponentNameInRegistry:  "example.com/secondref",
				ComponentNameInReference: "secondref",
				Version:                  "v0.0.3",
				ReferencedResourceName:   "secondjsonschemaref",
			}
			/*
				Short explanation of the setup:
				There are three components: base, firstRef, secondRef
				base contains two jsonschemas, a 'final' one and a 'referencing' one, the latter of which references the former one.
				firstRef contains two jsonschemas, which both reference jsonschemas from base
					- a simple one, which just references the (referencing) jsonschema from base
					- a complex one, which is a json object that within references both schemas from base
				secondRef contains a jsonschema which references the 'simple' jsonschema stored in firstRef

				So, in order to resolve the jsonschema from secondRef, one has to follow the references from secondRef to firstRef,
				then from firstRef to base and then from base[referencing] to base[final].
			*/

			// CREATE BASE COMPONENT
			blobfs := memoryfs.New()
			utils.ExpectNoError(blobfs.MkdirAll(blobPath, os.ModePerm))
			baseResourceName := "jsonschema"

			cdRes := []cdv2.Resource{
				createBlobResource(blobfs, baseResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schema.json", `
					{
						"type": "string"
					}
				`),
				createBlobResource(blobfs, baseConfig.ReferencedResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schemaref.json", fmt.Sprintf(`
					{
						"$ref": "cd://resources/%s"
					}
				`, baseResourceName)),
			}

			buildAndUploadComponentDescriptorWithArtifacts(ctx, testenv.Addr, baseConfig.ComponentNameInRegistry, baseConfig.Version, nil, cdRes, blobfs, ociClient, ociCache)

			// CREATE FIRST REFERENCING COMPONENT
			blobfs = memoryfs.New()
			utils.ExpectNoError(blobfs.MkdirAll(blobPath, os.ModePerm))
			refString := fmt.Sprintf("cd://componentReferences/%s/resources/%s", baseConfig.ComponentNameInReference, baseConfig.ReferencedResourceName)
			complexSchemaResourceName := "firstjsonschemarefcomplex"

			cdRes = []cdv2.Resource{
				createBlobResource(blobfs, firstRefConfig.ReferencedResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "plain.json", fmt.Sprintf(`
					{
						"$ref": "%s"
					}
				`, refString)),
				createBlobResource(blobfs, complexSchemaResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "complex.json", fmt.Sprintf(`
					{
						"type": "object",
						"properties": {
							"foo": {
								"$ref": "%s"
							},
							"bar": {
								"type": "array",
								"items": {
									"$ref": "%s"
								}
							},
							"baz": {
								"type": "number"
							}
						}
					}
				`, refString, fmt.Sprintf("cd://componentReferences/%s/resources/%s", baseConfig.ComponentNameInReference, baseResourceName))),
			}

			cdRef := []cdv2.ComponentReference{
				{
					Name:          baseConfig.ComponentNameInReference,
					ComponentName: baseConfig.ComponentNameInRegistry,
					Version:       baseConfig.Version,
				},
			}

			buildAndUploadComponentDescriptorWithArtifacts(ctx, testenv.Addr, firstRefConfig.ComponentNameInRegistry, firstRefConfig.Version, cdRef, cdRes, blobfs, ociClient, ociCache)

			// CREATE SECOND REFERENCING COMPONENT
			blobfs = memoryfs.New()
			utils.ExpectNoError(blobfs.MkdirAll(blobPath, os.ModePerm))

			cdRes = []cdv2.Resource{
				createBlobResource(blobfs, secondRefConfig.ReferencedResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schema2.json", fmt.Sprintf(`
					{
						"$ref": "cd://componentReferences/%s/resources/%s"
					}
				`, firstRefConfig.ComponentNameInReference, firstRefConfig.ReferencedResourceName)),
			}

			cdRef = []cdv2.ComponentReference{
				{
					Name:          firstRefConfig.ComponentNameInReference,
					ComponentName: firstRefConfig.ComponentNameInRegistry,
					Version:       firstRefConfig.Version,
				},
			}

			cd := buildAndUploadComponentDescriptorWithArtifacts(ctx, testenv.Addr, secondRefConfig.ComponentNameInRegistry, secondRefConfig.Version, cdRef, cdRes, blobfs, ociClient, ociCache)

			val := jsonschema.NewValidator(&jsonschema.ReferenceContext{
				ComponentDescriptor: cd,
				ComponentResolver:   cdoci.NewResolver(ociClient),
			})

			// this references the jsonschema resource in the secondRef component
			// it should resolve to {"type": "string"}
			testutils.ExpectNoError(val.CompileSchema([]byte(fmt.Sprintf(`
				{
					"$ref": "cd://resources/%s"
				}
			`, secondRefConfig.ReferencedResourceName))))

			pass := []byte(`"abc"`)
			fail := []byte("7")

			testutils.ExpectNoError(val.ValidateBytes(pass))
			Expect(val.ValidateBytes(fail)).NotTo(Succeed())

			// verify correct resolution of references nested in json structures
			// this references the 'complex' jsonschema resource from the firstRef component
			testutils.ExpectNoError(val.CompileSchema([]byte(fmt.Sprintf(`
				{
					"$ref": "cd://componentReferences/%s/resources/%s"
				}
			`, firstRefConfig.ComponentNameInReference, complexSchemaResourceName))))

			pass = []byte(`
				{
					"foo": "abc",
					"bar": [
						"def",
						"hij"
					],
					"baz": 7
				}
			`)
			testutils.ExpectNoError(val.ValidateBytes(pass))
			Expect(val.ValidateBytes([]byte(`
				{
					"foo": "abc",
					"bar": [
						"def",
						7
					],
					"baz": 7
				}
			`))).NotTo(Succeed())
		})

		It("should detect cyclic references", func() {
			blobPath := "blobs"
			blobfs := memoryfs.New()
			utils.ExpectNoError(blobfs.MkdirAll(blobPath, os.ModePerm))

			cycleConfig := &componentConfig{
				ComponentNameInRegistry:  "example.com/cyclicref",
				ComponentNameInReference: "cyclicref",
				Version:                  "v0.0.1",
				ReferencedResourceName:   "jsonschema",
			}
			relaySchemeResourceName := "relay"
			falsePositiveResourceName := "identical"

			// create referenced component
			cdRes := []cdv2.Resource{
				createBlobResource(blobfs, cycleConfig.ReferencedResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schema.json", fmt.Sprintf(`
					{
						"$ref": "cd://resources/%s"
					}
				`, cycleConfig.ReferencedResourceName)),
				createBlobResource(blobfs, falsePositiveResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schema2.json", `
					{
						"type": "string"
					}
				`),
				createBlobResource(blobfs, relaySchemeResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schema3.json", fmt.Sprintf(`
					{
						"$ref": "cd://resources/%s"
					}
				`, falsePositiveResourceName)),
			}

			buildAndUploadComponentDescriptorWithArtifacts(ctx, testenv.Addr, cycleConfig.ComponentNameInRegistry, cycleConfig.Version, nil, cdRes, blobfs, ociClient, ociCache)

			// create source component
			blobfs = memoryfs.New()
			utils.ExpectNoError(blobfs.MkdirAll(blobPath, os.ModePerm))

			cdRef := []cdv2.ComponentReference{
				{
					Name:          cycleConfig.ComponentNameInReference,
					ComponentName: cycleConfig.ComponentNameInRegistry,
					Version:       cycleConfig.Version,
				},
			}

			cdResSource := []cdv2.Resource{
				createBlobResource(blobfs, falsePositiveResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schema3.json", fmt.Sprintf(`
					{
						"$ref": "cd://componentReferences/%s/resources/%s"
					}
				`, cycleConfig.ComponentNameInReference, relaySchemeResourceName)),
			}

			cd := buildAndUploadComponentDescriptorWithArtifacts(ctx, testenv.Addr, "example.com/testcd", "v0.0.0", cdRef, cdResSource, blobfs, ociClient, ociCache)

			val := jsonschema.NewValidator(&jsonschema.ReferenceContext{
				ComponentDescriptor: cd,
				ComponentResolver:   cdoci.NewResolver(ociClient),
			})

			By("Test for false positives in cycle detection")
			/*
				Cycle detection works by checking whether a specific reference has already been resolved in the current run (which would indicate a cycle).
				A false positive could occur if a local reference A is resolved, which leads to a new reference into another component descriptor,
				where another reference B is found, which is identical to A. Both A and B look like "cd://resources/my_resource", but since they come from
				different CDs, both references actually refer to different jsonschemas and this is not a cycle and should not be detected as one.
			*/
			Expect(val.CompileSchema([]byte(fmt.Sprintf(`
				{
					"$ref": "cd://resources/%s"
				}
			`, falsePositiveResourceName)))).To(Succeed(), "false positive in cycle detection")

			By("Test for reference cycle detection")
			// this is somewhat ugly, since
			// 1. waiting for a timeout is not the best way to check for undetected cycles
			// 2. if this test fails, the looping function cannot be aborted and will continue to run until all goroutines are killed at the end of the test
			timeoutSeconds := 1
			ch := make(chan error, 1)
			go func() {
				err := val.CompileSchema([]byte(fmt.Sprintf(`
					{
						"$ref": "cd://componentReferences/%s/resources/%s"
					}
				`, cycleConfig.ComponentNameInReference, cycleConfig.ReferencedResourceName)))
				ch <- err
			}()
			select {
			case err := <-ch:
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cycle"))
			case <-time.After(time.Duration(timeoutSeconds) * time.Second):
				Fail(fmt.Sprintf("cyclic reference detection did not abort the fuction within %d seconds", timeoutSeconds), 0)
			}

		})

	})

})

// a small helper struct to better organize component references
type componentConfig struct {
	// how this component is called in the registry
	ComponentNameInRegistry string
	// how this component is called in references
	ComponentNameInReference string
	// version of this component
	Version string
	// how the resource which is referenced from outside is named
	ReferencedResourceName string
}

func buildAndUploadComponentDescriptorWithArtifacts(ctx context.Context, host, name, version string, cdRefs []cdv2.ComponentReference, cdRes []cdv2.Resource, fs vfs.FileSystem, ociClient ociclient.Client, ociCache cache.Cache) *cdv2.ComponentDescriptor {
	// define component descriptor
	cd := &cdv2.ComponentDescriptor{}

	cd.Name = name
	cd.Version = version
	cd.Provider = cdv2.InternalProvider
	cd.ComponentReferences = cdRefs
	cd.Resources = cdRes
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL:              fmt.Sprintf("%s/components/", host),
		ComponentNameMapping: cdv2.OCIRegistryURLPathMapping,
	}
	testutils.ExpectNoError(cdv2.InjectRepositoryContext(cd, &repoCtx))
	testutils.ExpectNoError(fs.MkdirAll("blobs", os.ModePerm))

	testutils.ExpectNoError(cdv2.DefaultComponent(cd))

	ca := ctf.NewComponentArchive(cd, fs)
	manifest, err := cdoci.NewManifestBuilder(ociCache, ca).Build(ctx)
	testutils.ExpectNoError(err)

	ref, err := cdoci.OCIRef(repoCtx, cd.Name, cd.Version)
	testutils.ExpectNoError(err)
	testutils.ExpectNoError(ociClient.PushManifest(ctx, ref, manifest))
	return cd
}

func buildLocalFilesystemResource(name, ttype, mediaType, path string) cdv2.Resource {
	res := cdv2.Resource{}
	res.Name = name
	res.Type = ttype
	res.Relation = cdv2.LocalRelation

	localFsAccess := cdv2.NewLocalFilesystemBlobAccess(path, mediaType)
	uAcc, err := cdv2.NewUnstructured(localFsAccess)
	testutils.ExpectNoError(err)
	res.Access = &uAcc
	return res
}

func createBlobResource(fs vfs.FileSystem, resourceName, resourceType, mediaType, blobPath, fileName, content string) cdv2.Resource {
	file, err := fs.Create(filepath.Join(blobPath, fileName))
	testutils.ExpectNoError(err)
	_, err = file.WriteString(content)
	utils.ExpectNoError(err)
	testutils.ExpectNoError(file.Close())
	return buildLocalFilesystemResource(resourceName, resourceType, mediaType, fileName)
}
