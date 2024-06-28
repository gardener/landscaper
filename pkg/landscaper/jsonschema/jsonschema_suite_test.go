// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	testutils2 "github.com/gardener/landscaper/pkg/components/testutils"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/landscaper/pkg/landscaper/jsonschema/testreg"

	apiconfig "github.com/gardener/landscaper/apis/config"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	testcred "github.com/gardener/component-cli/ociclient/credentials"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
	"github.com/gardener/landscaper/test/utils"
	testutils "github.com/gardener/landscaper/test/utils"
)

const jsonschemaResourceType = "landscaper.gardener.cloud/jsonschema"

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Test Suite")
}

var _ = Describe("jsonschema", func() {
	var (
		ctx  context.Context
		octx ocm.Context
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)
	})
	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
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

		It("should pass with a schema from a blueprint file reference and additional metadata", func() {
			localSchema := []byte(`{ "type": "string"}`)
			Expect(vfs.WriteFile(config.BlueprintFs, "myfile", localSchema, os.ModePerm)).To(Succeed())

			schemaBytes := []byte(`{ "title": "My Schema", "$ref": "blueprint://myfile", "description": "this is a schema"}`)
			data := []byte(`"valid"`)

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
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
			cd     *types.ComponentDescriptor
		)

		BeforeEach(func() {
			version := "v0.0.0"
			schemaBytes := []byte(`{ "type": "string"}`)

			// prepare a memory file system with 3 resources (json schemas)
			blobFs = memoryfs.New()
			Expect(blobFs.Mkdir(ctf.BlobsDirectoryName, os.ModePerm)).To(Succeed())

			// first resource: a json schema
			Expect(vfs.WriteFile(blobFs, ctf.BlobPath("default1.json"), schemaBytes, os.ModePerm)).To(Succeed())
			access1, err := testutils2.NewLocalFilesystemBlobAccess("default1.json", mediatype.JSONSchemaArtifactsMediaTypeV1)
			Expect(err).ToNot(HaveOccurred())
			resource1 := types.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Type:    mediatype.JSONSchemaType,
					Name:    "default",
					Version: version,
				},
				Relation: cdv2.LocalRelation,
				Access:   &access1,
			}

			// second resource: a compressed json schema
			var compressedSchema bytes.Buffer
			w := gzip.NewWriter(&compressedSchema)
			_, err = w.Write(schemaBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Close()).To(Succeed())
			Expect(vfs.WriteFile(blobFs, ctf.BlobPath("default2.json"), compressedSchema.Bytes(), os.ModePerm)).To(Succeed())
			access2, err := testutils2.NewLocalFilesystemBlobAccess("default2.json",
				mediatype.NewBuilder(mediatype.JSONSchemaArtifactsMediaTypeV1).Compression(mediatype.GZipCompression).Build().String())
			Expect(err).ToNot(HaveOccurred())
			resource2 := types.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Type:    mediatype.JSONSchemaType,
					Name:    "comp",
					Version: version,
				},
				Relation: cdv2.LocalRelation,
				Access:   &access2,
			}

			// third resource: like the first resource, but with an unknown mediatype
			Expect(vfs.WriteFile(blobFs, ctf.BlobPath("default3.json"), schemaBytes, os.ModePerm)).To(Succeed())
			access3, err := testutils2.NewLocalFilesystemBlobAccess("default3.json", "application/unknown")
			Expect(err).ToNot(HaveOccurred())
			resource3 := types.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Type:    mediatype.JSONSchemaType,
					Name:    "unknown",
					Version: version,
				},
				Relation: cdv2.LocalRelation,
				Access:   &access3,
			}

			// prepare component descriptor and registry
			repoCtx, err := testutils2.NewOCIRepositoryContext("example.com/reg")
			Expect(err).ToNot(HaveOccurred())

			cd = &types.ComponentDescriptor{
				Metadata: cdv2.Metadata{
					Version: "v2",
				},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta:          cdv2.ObjectMeta{Name: "example.com/test", Version: version},
					Provider:            "landscaper",
					RepositoryContexts:  []*types.UnstructuredTypedObject{&repoCtx},
					Resources:           []types.Resource{resource1, resource2, resource3},
					Sources:             []types.Source{},
					ComponentReferences: []types.ComponentReference{},
				},
			}

			blobResolver := testutils2.NewLocalFilesystemBlobResolver(blobFs)
			//TODO CONTEXTS: add repositoryContext to registry
			registryAccess, err := registries.GetFactory().CreateRegistryAccess(ctx, blobFs, nil, nil,
				&apiconfig.LocalRegistryConfiguration{RootPath: "./blobs"}, nil, cd, blobResolver)
			Expect(err).ToNot(HaveOccurred())

			// read component from registry
			componentVersion, err := registryAccess.GetComponentVersion(ctx, &types.ComponentVersionKey{
				Name:    cd.GetName(),
				Version: cd.GetVersion(),
			})
			Expect(err).To(Not(HaveOccurred()))

			config = &jsonschema.ReferenceContext{
				ComponentVersion: componentVersion,
				RegistryAccess:   registryAccess,
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
			schemaBytes := []byte(`{ "$ref": "cd://resources/comp"}`)
			data := []byte(`"valid"`)

			Expect(jsonschema.ValidateBytes(schemaBytes, data, config)).To(Succeed())
		})

		It("should throw an error if a wrong media type is used", func() {
			schemaBytes := []byte(`{ "$ref": "cd://resources/unknown"}`)
			data := []byte(`"valid"`)

			err := jsonschema.ValidateBytes(schemaBytes, data, config)
			Expect(err).ToNot(HaveOccurred())
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
			testenv        *testreg.Environment
			ociCache       cache.Cache
			ociClient      ociclient.Client
			registryAccess model.RegistryAccess
		)

		BeforeEach(func() {
			// create test registry
			testenv = testreg.New(testreg.Options{
				RegistryBinaryPath: filepath.Join("../../../", "bin", "registry"),
				Stdout:             GinkgoWriter,
				Stderr:             GinkgoWriter,
			})
			testutils.ExpectNoError(testenv.Start(ctx))
			testutils.ExpectNoError(testenv.WaitForRegistryToBeHealthy())

			keyring := testcred.New()
			testutils.ExpectNoError(keyring.AddAuthConfig(testenv.Addr, testcred.AuthConfig{
				Username: testenv.BasicAuth.Username,
				Password: testenv.BasicAuth.Password,
			}))

			var err error
			ociCache, err = cache.NewCache(logging.Discard().Logr())
			testutils.ExpectNoError(err)
			ociClient, err = ociclient.NewClient(logging.Discard().Logr(), ociclient.WithKeyring(keyring), ociclient.WithCache(ociCache))
			testutils.ExpectNoError(err)

			fs := memoryfs.New()

			Expect(fs.MkdirAll("testdata", 0o777)).To(Succeed())
			base64creds := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`%s:%s`, testenv.BasicAuth.Username, testenv.BasicAuth.Password)))
			dockerconfig := []byte(fmt.Sprintf(`{"auths": {"%s": {"auth": "%s"}}}`, testenv.Addr, base64creds))
			f, err := fs.OpenFile(filepath.Join("testdata", "dockerconfig.json"), os.O_CREATE|os.O_RDWR, 0o777)
			Expect(err).ToNot(HaveOccurred())
			cnt, err := f.Write(dockerconfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(cnt).ToNot(Equal(0))
			f.Close()

			ociconfig := &apiconfig.OCIConfiguration{
				ConfigFiles: []string{"testdata/dockerconfig.json"},
			}

			config := []byte(fmt.Sprintf(`{
	"type": "credentials.config.ocm.software",
    "consumers": [
        {
            "identity": {
                "type": "OCIRegistry",
                "hostname": %q,
                "port": %q
            },
            "credentials": [
                {
                    "type": "Credentials",
                    "properties": {
                        "username": %q,
                        "password": %q,
                        "certificateAuthority": %q
                    }
                }
            ]
        }
    ]
}
`, strings.Split(testenv.Addr, ":")[0], strings.Split(testenv.Addr, ":")[1], testenv.BasicAuth.Username, testenv.BasicAuth.Password, testenv.Certificate.CA))
			secrets := []corev1.Secret{{
				Data: map[string][]byte{`.ocmcredentialconfig`: config}}}
			registryAccess, err = registries.GetFactory().CreateRegistryAccess(ctx, fs, nil, secrets, nil,
				ociconfig, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			testutils.ExpectNoError(testenv.Close())
			ctx.Done()
		})

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

			cdRes := []types.Resource{
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

			cdRes = []types.Resource{
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

			cdRef := []types.ComponentReference{
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

			cdRes = []types.Resource{
				createBlobResource(blobfs, secondRefConfig.ReferencedResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schema2.json", fmt.Sprintf(`
					{
						"$ref": "cd://componentReferences/%s/resources/%s"
					}
				`, firstRefConfig.ComponentNameInReference, firstRefConfig.ReferencedResourceName)),
			}

			cdRef = []types.ComponentReference{
				{
					Name:          firstRefConfig.ComponentNameInReference,
					ComponentName: firstRefConfig.ComponentNameInRegistry,
					Version:       firstRefConfig.Version,
				},
			}

			cd := buildAndUploadComponentDescriptorWithArtifacts(ctx, testenv.Addr, secondRefConfig.ComponentNameInRegistry, secondRefConfig.Version, cdRef, cdRes, blobfs, ociClient, ociCache)

			//TODO CONTEXTS: add repositoryContext cd.GetEffectiveRepositoryContext() to registry
			secondComponentVersion, err := registryAccess.GetComponentVersion(ctx, &types.ComponentVersionKey{
				Name:    cd.GetName(),
				Version: cd.GetVersion(),
			})
			Expect(err).NotTo(HaveOccurred())

			val := jsonschema.NewValidator(&jsonschema.ReferenceContext{
				ComponentVersion: secondComponentVersion,
				RegistryAccess:   registryAccess,
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
			cdRes := []types.Resource{
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

			cdRef := []types.ComponentReference{
				{
					Name:          cycleConfig.ComponentNameInReference,
					ComponentName: cycleConfig.ComponentNameInRegistry,
					Version:       cycleConfig.Version,
				},
			}

			cdResSource := []types.Resource{
				createBlobResource(blobfs, falsePositiveResourceName, jsonschemaResourceType, mediatype.JSONSchemaArtifactsMediaTypeV1, blobPath, "schema3.json", fmt.Sprintf(`
					{
						"$ref": "cd://componentReferences/%s/resources/%s"
					}
				`, cycleConfig.ComponentNameInReference, relaySchemeResourceName)),
			}

			cd := buildAndUploadComponentDescriptorWithArtifacts(ctx, testenv.Addr, "example.com/testcd", "v0.0.0", cdRef, cdResSource, blobfs, ociClient, ociCache)

			//TODO CONTEXTS: add repositoryContext cd.GetEffectiveRepositoryContext() to registry
			componentVersion, err := registryAccess.GetComponentVersion(ctx, &types.ComponentVersionKey{
				Name:    cd.GetName(),
				Version: cd.GetVersion(),
			})
			Expect(err).NotTo(HaveOccurred())

			val := jsonschema.NewValidator(&jsonschema.ReferenceContext{
				ComponentVersion: componentVersion,
				RegistryAccess:   registryAccess,
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

	Context("WithLocalRegistry", func() {

		var (
			registryAccess      model.RegistryAccess
			componentVersion    model.ComponentVersion
			repositoryContext   types.UnstructuredTypedObject
			localregistryconfig *apiconfig.LocalRegistryConfiguration
		)

		BeforeEach(func() {
			var err error

			localregistryconfig := &apiconfig.LocalRegistryConfiguration{RootPath: "./testdata/registry"}
			registryAccess, err = registries.GetFactory().CreateRegistryAccess(ctx, nil, nil, nil,
				localregistryconfig, nil, nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(repositoryContext.UnmarshalJSON([]byte(`{"type":"local"}`))).To(Succeed())

			//TODO CONTEXTS: add repositoryContext repositoryContext to registry
			componentVersion, err = registryAccess.GetComponentVersion(ctx, &types.ComponentVersionKey{
				Name:    "example.com/root",
				Version: "v0.1.0",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(componentVersion).ToNot(BeNil())
		})

		It("should resolve with explicit repository context", func() {
			registryAccess, err := registries.GetFactory().CreateRegistryAccess(ctx, nil, nil, nil,
				localregistryconfig, nil, nil)
			Expect(err).ToNot(HaveOccurred())

			referenceResolver := jsonschema.NewReferenceResolver(&jsonschema.ReferenceContext{
				ComponentVersion:  componentVersion,
				RegistryAccess:    registryAccess,
				RepositoryContext: &repositoryContext,
			})

			resolved, err := referenceResolver.Resolve([]byte(`
			{
				"$ref": "cd://componentReferences/ref-1/resources/resourcesschema"
			}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).ToNot(BeNil())
		})

		It("should not resolve without explicit repository context", func() {
			registryAccess, err := registries.GetFactory().CreateRegistryAccess(ctx, nil, nil, nil,
				localregistryconfig, nil, nil)
			Expect(err).ToNot(HaveOccurred())

			referenceResolver := jsonschema.NewReferenceResolver(&jsonschema.ReferenceContext{
				ComponentVersion: componentVersion,
				RegistryAccess:   registryAccess,
			})

			resolved, err := referenceResolver.Resolve([]byte(`
			{
				"$ref": "cd://componentReferences/ref-1/resources/resourcesschema"
			}`))
			Expect(err).To(HaveOccurred())
			Expect(resolved).To(BeNil())
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

func buildAndUploadComponentDescriptorWithArtifacts(ctx context.Context, host, name, version string, cdRefs []types.ComponentReference, cdRes []types.Resource, fs vfs.FileSystem, ociClient ociclient.Client, ociCache cache.Cache) *types.ComponentDescriptor {
	// define component descriptor
	cd := &types.ComponentDescriptor{}

	cd.Name = name
	cd.Version = version
	cd.Provider = "internal"
	cd.ComponentReferences = cdRefs
	cd.Resources = cdRes
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL:              host,
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

func buildLocalFilesystemResource(name, ttype, mediaType, path string) types.Resource {
	res := types.Resource{}
	res.Name = name
	res.Type = ttype
	res.Relation = cdv2.LocalRelation

	localFsAccess, err := testutils2.NewLocalFilesystemBlobAccess(path, mediaType)
	testutils.ExpectNoError(err)
	res.Access = &localFsAccess
	return res
}

func createBlobResource(fs vfs.FileSystem, resourceName, resourceType, mediaType, blobPath, fileName, content string) types.Resource {
	file, err := fs.Create(filepath.Join(blobPath, fileName))
	testutils.ExpectNoError(err)
	_, err = file.WriteString(content)
	utils.ExpectNoError(err)
	testutils.ExpectNoError(file.Close())
	return buildLocalFilesystemResource(resourceName, resourceType, mediaType, fileName)
}
