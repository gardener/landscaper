// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/cnudie/componentresolvers"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/testutils"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Executions Test Suite")
}

var _ = Describe("TemplateDeployExecutions", func() {

	sharedTestdataDir := filepath.Join("./testdata", "shared_data")

	Context("GoTemplate", func() {
		testdataDir := filepath.Join("./testdata", "gotemplate")
		runTestSuite(testdataDir, sharedTestdataDir)
		runTestSuiteGoTemplate(testdataDir, sharedTestdataDir)
	})

	Context("Spiff", func() {
		testdataDir := filepath.Join("./testdata", "spifftemplate")
		testdataDirYAML := filepath.Join(testdataDir, "yaml")
		testdataDirText := filepath.Join(testdataDir, "text")
		Context("YAML", func() {
			runTestSuite(testdataDirYAML, sharedTestdataDir)
			runTestSuiteSpiff(testdataDirYAML, sharedTestdataDir)
		})
		Context("Text", func() {
			runTestSuite(testdataDirText, sharedTestdataDir)
			runTestSuiteSpiff(testdataDirText, sharedTestdataDir)
		})
	})

})

func runTestSuite(testdataDir, sharedTestdataDir string) {
	var stateHandler template.GenericStateHandler

	BeforeEach(func() {
		stateHandler = template.NewMemoryStateHandler()
	})

	Context("TemplateSubinstallationExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-20.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.SubinstallationExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			res, err := op.TemplateSubinstallationExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil, nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			Expect(res[0].Name).To(Equal("my-subinstallation"))
			Expect(res[0].Blueprint).To(MatchFields(IgnoreExtras, Fields{
				"Ref": Equal("cd://resources/myblueprint"),
			}))
		})

		It("should use imports to template installations", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-21.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.SubinstallationExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			res, err := op.TemplateSubinstallationExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil,
					map[string]interface{}{"blueprintName": "some-blueprint-name"})))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			Expect(res[0].Name).To(Equal("my-subinstallation"))
			Expect(res[0].Blueprint).To(MatchFields(IgnoreExtras, Fields{
				"Ref": Equal("cd://resources/some-blueprint-name"),
			}))
		})
	})

	Context("TemplateDeployExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-01.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil, nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			Expect(res[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name": Equal("init"),
				"Type": Equal(core.DeployItemType("container")),
			}))
		})

		It("should use the import values to template", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-02.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil,
					map[string]interface{}{"version": "0.0.0"})))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should read the content of a file to template", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-03.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			memFs := memoryfs.New()
			err = vfs.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: memFs}, nil, nil, nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should use a resource from the component descriptor", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-04.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			imageAccess, err := componentresolvers.NewOCIRegistryAccess("quay.io/example/myimage:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			cd := &types.ComponentDescriptor{
				Metadata: types.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/mycomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []*types.UnstructuredTypedObject{},
					Provider:           cdv2.InternalProvider,
					Resources: []types.Resource{
						{
							IdentityObjectMeta: cdv2.IdentityObjectMeta{
								Name:    "mycustomimage",
								Version: "1.0.0",
								Type:    cdv2.OCIImageType,
							},
							Relation: cdv2.ExternalRelation,
							Access:   &imageAccess,
						},
					},
				},
			}
			Expect(cdv2.DefaultComponent(cd)).To(Succeed())
			componentVersion := testutils.NewTestComponentVersionFromReader(cd, nil, nil)

			res, err := op.TemplateDeployExecutions(
				template.NewDeployExecutionOptions(
					template.NewBlueprintExecutionOptions(
						nil,
						&blueprints.Blueprint{Info: blue, Fs: nil},
						componentVersion,
						nil,
						nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "quay.io/example/myimage:1.0.0"))
		})

		It("should use a resource from the component descriptor's referenced component", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-10.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			imageAccess, err := componentresolvers.NewOCIRegistryAccess("quay.io/example/myimage:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			cd := &types.ComponentDescriptor{
				Metadata: types.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/mycomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []*types.UnstructuredTypedObject{},
					Provider:           cdv2.InternalProvider,
					ComponentReferences: []types.ComponentReference{
						{
							Name:          "my-referenced-component",
							ComponentName: "example.com/myrefcomp",
							Version:       "1.0.0",
						},
					},
				},
			}
			Expect(cdv2.DefaultComponent(cd)).To(Succeed())
			componentVersion := testutils.NewTestComponentVersionFromReader(cd, nil, nil)

			cd2 := types.ComponentDescriptor{
				Metadata: types.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/myrefcomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []*types.UnstructuredTypedObject{},
					Provider:           cdv2.InternalProvider,
					Resources: []types.Resource{
						{
							IdentityObjectMeta: cdv2.IdentityObjectMeta{
								Name:    "ubuntu",
								Version: "1.0.0",
								Type:    cdv2.OCIImageType,
							},
							Relation: cdv2.ExternalRelation,
							Access:   &imageAccess,
						},
					},
				},
			}
			Expect(cdv2.DefaultComponent(&cd2)).To(Succeed())
			componentVersion2 := testutils.NewTestComponentVersionFromReader(&cd2, nil, nil)

			componentVersionList := &model.ComponentVersionList{
				Components: []model.ComponentVersion{
					componentVersion2,
				},
			}

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, componentVersion, componentVersionList, nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "quay.io/example/myimage:1.0.0"))
		})

		It("should throw an error when the template tries to template a undefined value", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-08.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			_, err = op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil,
					map[string]interface{}{"version": "0.0.0"})))
			Expect(err).To(HaveOccurred())
		})

		It("should use the state to template", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-09.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			_, err = op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil,
					map[string]interface{}{"version": "0.0.1"})))
			Expect(err).ToNot(HaveOccurred())

			state := map[string]string{
				"version": "0.0.2",
			}
			stateBytes, err := json.Marshal(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(stateHandler.Store(context.TODO(), "deploy"+exec[0].Name, stateBytes)).To(Succeed())
			_, err = op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil,
					map[string]interface{}{"version": "0.0.2"})))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should use the blueprint information", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-11.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			componentDef := lsv1alpha1.ComponentDescriptorDefinition{}
			componentDef.Reference = &lsv1alpha1.ComponentDescriptorReference{}
			componentDef.Reference.ComponentName = "my-comp"
			componentDef.Reference.Version = "0.1.0"

			blueprintDef := lsv1alpha1.BlueprintDefinition{}
			blueprintDef.Reference = &lsv1alpha1.RemoteBlueprintReference{}
			blueprintDef.Reference.ResourceName = "my-res"

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(
					&lsv1alpha1.Installation{
						Spec: lsv1alpha1.InstallationSpec{
							Blueprint:           blueprintDef,
							ComponentDescriptor: &componentDef,
						},
					},
					&blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil, nil)))

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

		It("should generate an image vector", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-12.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			cdRaw, err := os.ReadFile(filepath.Join(sharedTestdataDir, "component-descriptor-12.yaml"))
			Expect(err).ToNot(HaveOccurred())
			cd := &types.ComponentDescriptor{}
			Expect(yaml.Unmarshal(cdRaw, cd)).ToNot(HaveOccurred())
			Expect(cdv2.DefaultComponent(cd)).To(Succeed())
			componentVersion := testutils.NewTestComponentVersionFromReader(cd, nil, nil)

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(
					nil,
					&blueprints.Blueprint{Info: blue, Fs: nil},
					componentVersion,
					&model.ComponentVersionList{},
					nil)))

			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())

			result, err := os.ReadFile(filepath.Join(sharedTestdataDir, "result-12.yaml"))
			Expect(err).ToNot(HaveOccurred())
			resultString := string(result)

			entries := []string{"imageVectorOverWrite1", "imageVectorOverWrite2", "imageVectorOverWrite3"}

			for _, nextEntry := range entries {
				imageMap, ok := config[nextEntry].(map[string]interface{})
				Expect(ok).To(BeTrue())
				imageVector, err := yaml.Marshal(imageMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(imageVector)).To(BeIdenticalTo(resultString))
			}
		})

		It("should use a parsed oci ref to template", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-13.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil,
					map[string]interface{}{
						"ref1": "myimage:0.0.0",
						"ref2": "myimage@sha256:66371f17cc61bbbed2667b0285a10981deba5eb969df9bfd4cf273706044ddcb",
					})))

			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image0", "myimage:0.0.0"))
			Expect(config).To(HaveKeyWithValue("image1", "myimage:0.0.0"))
			Expect(config).To(HaveKeyWithValue("image2", "myimage@sha256:66371f17cc61bbbed2667b0285a10981deba5eb969df9bfd4cf273706044ddcb"))
		})
	})

	Context("TemplateExportExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-05.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			res, err := op.TemplateExportExecutions(template.NewExportExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil, nil), nil))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKeyWithValue("testKey", "myval"))
		})

		It("should use the export values to template", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-06.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			res, err := op.TemplateExportExecutions(template.NewExportExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil, nil),
				map[string]interface{}{"version": "0.0.0"}))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should read the content of a file to template", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-07.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			memFs := memoryfs.New()
			err = vfs.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			res, err := op.TemplateExportExecutions(template.NewExportExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: memFs}, nil, nil, nil), nil))

			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			Expect(res).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})
	})
}

func runTestSuiteGoTemplate(testdataDir, sharedTestdataDir string) {
	var (
		stateHandler template.GenericStateHandler

		executeTemplate = func(templateFile string, imports map[string]interface{}) ([]template.DeployItemSpecification, error) {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, templateFile))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec

			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			cdRaw, err := os.ReadFile(filepath.Join(sharedTestdataDir, "component-descriptor-12.yaml"))
			Expect(err).ToNot(HaveOccurred())
			cd := &types.ComponentDescriptor{}
			Expect(yaml.Unmarshal(cdRaw, cd)).ToNot(HaveOccurred())
			Expect(cdv2.DefaultComponent(cd)).To(Succeed())
			componentVersion := testutils.NewTestComponentVersionFromReader(cd, nil, nil)

			res, err := op.TemplateDeployExecutions(
				template.NewDeployExecutionOptions(
					template.NewBlueprintExecutionOptions(
						nil,
						&blueprints.Blueprint{Info: blue, Fs: nil},
						componentVersion,
						&model.ComponentVersionList{},
						imports)))

			return res, err
		}
	)

	BeforeEach(func() {
		stateHandler = template.NewMemoryStateHandler()
	})

	Context("Error Messages", func() {
		It("should handle template execution errors", func() {
			res, err := executeTemplate("template-22.yaml", map[string]interface{}{
				"config": map[string]interface{}{
					"verbosity": 10,
					"memory": map[string]interface{}{
						"min": 128,
						"max": 1024,
					},
					"cert": "abcdef1234567",
					"image": map[string]interface{}{
						"name":    "test",
						"version": "0.0.1",
					},
				},
			})

			Expect(err).To(HaveOccurred())
			Expect(res).To(BeNil())

			errstr := err.Error()

			Expect(errstr).To(ContainSubstring(`template source:
2:    - name: init
3:      type: manifest
4:      config:
5:        apiVersion: example.test/v1
6:        kind: Configuration
7:        verbosity: {{ .invalid.verbosity }}
                                ˆ≈≈≈≈≈≈≈
8:        memory:
9:          min: {{ .imports.config.memory.min }}
10:         max: {{ .imports.config.memory.max }}
11:       cert:
12:         {{ .imports.config.cert }}`))

			Expect(errstr).To(ContainSubstring(`imports: {"config":{"cert":"[...] (string)","image":{"name":"[...] (string)","version":"[...] (string)"},"memory":{"max":"[...] (int)","min":"[...] (int)"},"verbosity":"[...] (int)"}}`))
			Expect(errstr).To(ContainSubstring("cd:"))
			Expect(errstr).To(ContainSubstring("components:"))
			Expect(errstr).To(ContainSubstring("state:"))
		})

		It("should handle empty values", func() {
			res, err := executeTemplate("template-23.yaml", map[string]interface{}{
				"config": map[string]interface{}{
					"memory": map[string]interface{}{
						"min": 128,
					},
					"cert": "abcdef1234567",
					"image": map[string]interface{}{
						"name":    "test",
						"version": "0.0.1",
					},
				},
			})

			Expect(err).To(HaveOccurred())
			Expect(res).To(BeNil())

			errstr := err.Error()
			Expect(errstr).To(ContainSubstring(`template "deploy execution" contains fields with "no value":`))

			Expect(errstr).To(ContainSubstring(`line 7:15
2:    - name: init
3:      type: manifest
4:      config:
5:        apiVersion: example.test/v1
6:        kind: Configuration
7:        verbosity: <no value>
                     ˆ≈≈≈≈≈≈≈
8:        memory:
9:          min: 128
10:         max: <no value>
11:       cert:
12:         abcdef1234567

line 10:11
5:        apiVersion: example.test/v1
6:        kind: Configuration
7:        verbosity: <no value>
8:        memory:
9:          min: 128
10:         max: <no value>
                 ˆ≈≈≈≈≈≈≈
11:       cert:
12:         abcdef1234567
13:       image: test:0.0.1
14:   `))

			Expect(errstr).To(ContainSubstring("imports:"))
			Expect(errstr).To(ContainSubstring(`{"config":{"cert":"[...] (string)","image":{"name":"[...] (string)","version":"[...] (string)"},"memory":{"min":"[...] (int)"}}}`))

			Expect(errstr).To(ContainSubstring("cd:"))
			Expect(errstr).To(ContainSubstring("components:"))
			Expect(errstr).To(ContainSubstring("state:"))
		})

		It("should handle template parsing errors", func() {
			res, err := executeTemplate("template-24.yaml", map[string]interface{}{
				"config": map[string]interface{}{
					"verbosity": 10,
					"memory": map[string]interface{}{
						"min": 128,
						"max": 1024,
					},
					"cert": "abcdef1234567",
					"image": map[string]interface{}{
						"name":    "test",
						"version": "0.0.1",
					},
				},
			})

			Expect(err).To(HaveOccurred())
			Expect(res).To(BeNil())

			errstr := err.Error()
			Expect(errstr).To(ContainSubstring(`template source:
3:      type: manifest
4:      config:
5:        apiVersion: example.test/v1
6:        kind: Configuration
7:        memory:
8:          min: {{ .imports.config.memory.min }
      ˆ≈≈≈≈≈≈≈
9:          max: {{ .imports.config.memory.max }}
10:       cert:
11:         {{ .imports.config.cert }}
12:       image: {{ ( print .imports.config.image.name ":" .imports.config.image.version ) }}
13:   `))
		})
	})
}

func runTestSuiteSpiff(testdataDir, sharedTestdataDir string) {
	var stateHandler template.GenericStateHandler

	BeforeEach(func() {
		stateHandler = template.NewMemoryStateHandler()
	})

	Context("Error Messages", func() {
		It("should handle template execution errors", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-22.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			blue.Imports = lsv1alpha1.ImportDefinitionList{
				{
					FieldValueDefinition: lsv1alpha1.FieldValueDefinition{
						Name: "config",
					},
					Type: "object",
				},
			}
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler))

			cdRaw, err := os.ReadFile(filepath.Join(sharedTestdataDir, "component-descriptor-12.yaml"))
			Expect(err).ToNot(HaveOccurred())
			cd := &types.ComponentDescriptor{}
			Expect(yaml.Unmarshal(cdRaw, cd)).ToNot(HaveOccurred())
			Expect(cdv2.DefaultComponent(cd)).To(Succeed())
			componentVersion := testutils.NewTestComponentVersionFromReader(cd, nil, nil)

			res, err := op.TemplateDeployExecutions(
				template.NewDeployExecutionOptions(
					template.NewBlueprintExecutionOptions(
						nil,
						&blueprints.Blueprint{Info: blue, Fs: nil},
						componentVersion,
						&model.ComponentVersionList{},
						map[string]interface{}{
							"config": map[string]interface{}{
								"verbosity": 10,
								"memory": map[string]interface{}{
									"min": 128,
									"max": 1024,
								},
								"cert": "abcdef1234567",
								"image": map[string]interface{}{
									"name":    "test",
									"version": "0.0.1",
								},
							},
						})))

			Expect(err).To(HaveOccurred())
			Expect(res).To(BeNil())

			errstr := err.Error()

			Expect(errstr).To(ContainSubstring("imports:"))
			Expect(errstr).To(ContainSubstring(`{"config":{"cert":"[...] (string)","image":{"name":"[...] (string)","version":"[...] (string)"},"memory":{"max":"[...] (int)","min":"[...] (int)"},"verbosity":"[...] (int)"}}`))

			Expect(errstr).To(ContainSubstring("cd:"))
			Expect(errstr).To(ContainSubstring("components:"))
		})
	})
}
