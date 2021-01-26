// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/apis/v2/cdutils"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Executions Test Suite")
}

var _ = Describe("TemplateDeployExecutions", func() {

	Context("GoTemplate", func() {
		testdataDir := filepath.Join("./testdata", "gotemplate")
		runTestSuite(testdataDir)
	})

	Context("Spiff", func() {
		testdataDir := filepath.Join("./testdata", "spifftemplate")
		runTestSuite(testdataDir)
	})

})

func runTestSuite(testdataDir string) {
	var stateHandler GenericStateHandler

	BeforeEach(func() {
		stateHandler = NewMemoryStateHandler()
	})

	Context("TemplateDeployExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-01.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(nil, stateHandler)

			res, err := op.TemplateDeployExecutions(DeployExecutionOptions{
				Blueprint: &blueprints.Blueprint{
					Info: blue,
					Fs:   nil,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			Expect(res[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name": Equal("init"),
				"Type": Equal(lsv1alpha1.DeployItemType("container")),
			}))
		})

		It("should use the import values to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-02.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(nil, stateHandler)

			res, err := op.TemplateDeployExecutions(DeployExecutionOptions{
				Imports: map[string]interface{}{"version": "0.0.0"},
				Blueprint: &blueprints.Blueprint{
					Info: blue,
					Fs:   nil,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should read the content of a file to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-03.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(nil, stateHandler)

			memFs := memoryfs.New()
			err = vfs.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			res, err := op.TemplateDeployExecutions(DeployExecutionOptions{
				Blueprint: &blueprints.Blueprint{
					Info: blue,
					Fs:   memFs,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should use a resource from the component descriptor", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-04.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(nil, stateHandler)

			imageAccess, err := cdutils.ToUnstructuredTypedObject(cdv2.DefaultJSONTypedObjectCodec, &cdv2.OCIRegistryAccess{
				ObjectType: cdv2.ObjectType{
					Type: cdv2.OCIRegistryType,
				},
				ImageReference: "quay.io/example/myimage:1.0.0",
			})
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{
				Metadata: cdv2.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/mycomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []cdv2.RepositoryContext{},
					Provider:           cdv2.InternalProvider,
					Resources: []cdv2.Resource{
						{
							IdentityObjectMeta: cdv2.IdentityObjectMeta{
								Name:    "mycustomimage",
								Version: "1.0.0",
								Type:    cdv2.OCIImageType,
							},
							Relation: cdv2.ExternalRelation,
							Access:   imageAccess,
						},
					},
				},
			}
			Expect(cdv2.DefaultComponent(cd)).To(Succeed())

			res, err := op.TemplateDeployExecutions(DeployExecutionOptions{
				Blueprint: &blueprints.Blueprint{
					Info: blue,
				},
				ComponentDescriptor: cd,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "quay.io/example/myimage:1.0.0"))
		})

		It("should use a resource from the component descriptor's referenced component", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-10.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(nil, stateHandler)

			imageAccess, err := cdutils.ToUnstructuredTypedObject(cdv2.DefaultJSONTypedObjectCodec, &cdv2.OCIRegistryAccess{
				ObjectType: cdv2.ObjectType{
					Type: cdv2.OCIRegistryType,
				},
				ImageReference: "quay.io/example/myimage:1.0.0",
			})
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{
				Metadata: cdv2.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/mycomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []cdv2.RepositoryContext{},
					Provider:           cdv2.InternalProvider,
					ComponentReferences: []cdv2.ComponentReference{
						{
							Name:          "my-referenced-component",
							ComponentName: "example.com/myrefcomp",
							Version:       "1.0.0",
						},
					},
				},
			}
			Expect(cdv2.DefaultComponent(cd)).To(Succeed())

			cd2 := cdv2.ComponentDescriptor{
				Metadata: cdv2.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/myrefcomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []cdv2.RepositoryContext{},
					Provider:           cdv2.InternalProvider,
					Resources: []cdv2.Resource{
						{
							IdentityObjectMeta: cdv2.IdentityObjectMeta{
								Name:    "ubuntu",
								Version: "1.0.0",
								Type:    cdv2.OCIImageType,
							},
							Relation: cdv2.ExternalRelation,
							Access:   imageAccess,
						},
					},
				},
			}
			Expect(cdv2.DefaultComponent(&cd2)).To(Succeed())
			list := &cdv2.ComponentDescriptorList{
				Metadata:   cdv2.Metadata{},
				Components: []cdv2.ComponentDescriptor{cd2},
			}

			res, err := op.TemplateDeployExecutions(DeployExecutionOptions{
				Blueprint: &blueprints.Blueprint{
					Info: blue,
				},
				ComponentDescriptor:  cd,
				ComponentDescriptors: list,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "quay.io/example/myimage:1.0.0"))
		})

		It("should throw an error when the template tries to template a undefined value", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-08.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(nil, stateHandler)

			_, err = op.TemplateDeployExecutions(DeployExecutionOptions{
				Imports: map[string]interface{}{"version": "0.0.0"},
				Blueprint: &blueprints.Blueprint{
					Info: blue,
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("should use the state to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-09.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(nil, stateHandler)

			_, err = op.TemplateDeployExecutions(DeployExecutionOptions{
				Imports: map[string]interface{}{"version": "0.0.1"},
				Blueprint: &blueprints.Blueprint{
					Info: blue,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			state := map[string]string{
				"version": "0.0.2",
			}
			stateBytes, err := json.Marshal(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(stateHandler.Store(context.TODO(), exec[0].Name, stateBytes)).To(Succeed())
			_, err = op.TemplateDeployExecutions(DeployExecutionOptions{
				Imports: map[string]interface{}{"version": "0.0.2"},
				Blueprint: &blueprints.Blueprint{
					Info: blue,
				},
			})

			Expect(err).ToNot(HaveOccurred())
		})

		It("should use the blueprint information", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-11.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(nil, stateHandler)

			componentDef := lsv1alpha1.ComponentDescriptorDefinition{}
			componentDef.Reference = &lsv1alpha1.ComponentDescriptorReference{}
			componentDef.Reference.ComponentName = "my-comp"
			componentDef.Reference.Version = "0.1.0"

			blueprintDef := lsv1alpha1.BlueprintDefinition{}
			blueprintDef.Reference = &lsv1alpha1.RemoteBlueprintReference{}
			blueprintDef.Reference.ResourceName = "my-res"

			res, err := op.TemplateDeployExecutions(DeployExecutionOptions{
				Installation: &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Blueprint:           blueprintDef,
						ComponentDescriptor: &componentDef,
					},
				},
				Blueprint: &blueprints.Blueprint{
					Info: blue,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(MatchKeys(IgnoreExtras, Keys{
				"blueprint": MatchAllKeys(Keys{
					"ref": MatchAllKeys(Keys{
						"resourceName": Equal("my-res"),
					}),
				}),
				"componentDescriptor": MatchAllKeys(Keys{
					"ref": MatchKeys(IgnoreExtras, Keys{
						"componentName": Equal("my-comp"),
						"version":       Equal("0.1.0"),
					}),
				}),
			}))
		})
	})

	Context("TemplateExportExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-05.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := New(nil, stateHandler)

			res, err := op.TemplateExportExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKeyWithValue("testKey", "myval"))
		})

		It("should use the export values to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-06.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := New(nil, stateHandler)

			res, err := op.TemplateExportExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, map[string]interface{}{"version": "0.0.0"})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should read the content of a file to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-07.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := New(nil, stateHandler)

			memFs := memoryfs.New()
			err = vfs.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			res, err := op.TemplateExportExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   memFs,
			}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			Expect(res).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})
	})
}
