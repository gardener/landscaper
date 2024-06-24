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
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"
	"sigs.k8s.io/yaml"

	apiconfig "github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/components/testutils"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/common"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"
	"github.com/gardener/landscaper/pkg/utils/blueprints"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Executions Test Suite")
}

var _ = Describe("TemplateDeployExecutions", func() {
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

	sharedTestdataDir := filepath.Join("./testdata", "shared_data")

	Context("GoTemplate", func() {
		testdataDir := filepath.Join("./testdata", "gotemplate")
		runTestSuite(testdataDir, sharedTestdataDir)
		runTestSuiteGoTemplate(testdataDir)
	})

	Context("Spiff", func() {
		testdataDir := filepath.Join("./testdata", "spifftemplate")
		testdataDirYAML := filepath.Join(testdataDir, "yaml")
		testdataDirText := filepath.Join(testdataDir, "text")
		Context("YAML", func() {
			runTestSuite(testdataDirYAML, sharedTestdataDir)
			runTestSuiteSpiff(testdataDirYAML)
		})
		Context("Text", func() {
			runTestSuite(testdataDirText, sharedTestdataDir)
			runTestSuiteSpiff(testdataDirText)
		})
	})

})

func runTestSuite(testdataDir, sharedTestdataDir string) {
	var (
		stateHandler template.GenericStateHandler
		ctx          context.Context
		octx         ocm.Context
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)

		stateHandler = template.NewMemoryStateHandler()
	})

	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	Context("TemplateSubinstallationExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-20.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.SubinstallationExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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

		DescribeTable("should use a resource from the component descriptor", func(ocmSchemaVersion string) {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-04.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.Annotations = map[string]string{common.OCM_SCHEMA_VERSION: ocmSchemaVersion}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			imageAccess1, err := testutils.NewOCIRegistryAccess("quay.io/example/myimage:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			imageAccess2, err := testutils.NewOCIRegistryAccess("quay.io/example/yourimage:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			cd := &types.ComponentDescriptor{
				Metadata: types.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/mycomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []*types.UnstructuredTypedObject{},
					Provider:           "internal",
					Resources: []types.Resource{
						{
							IdentityObjectMeta: cdv2.IdentityObjectMeta{
								Name:          "mycustomimage",
								Version:       "1.0.0",
								Type:          cdv2.OCIImageType,
								ExtraIdentity: cdv2.Identity{"class": "image"},
							},
							Relation: cdv2.ExternalRelation,
							Access:   &imageAccess1,
						},
						{
							IdentityObjectMeta: cdv2.IdentityObjectMeta{
								Name:          "yourcustomimage",
								Version:       "1.0.0",
								Type:          cdv2.OCIImageType,
								ExtraIdentity: cdv2.Identity{"class": "image"},
							},
							Relation: cdv2.ExternalRelation,
							Access:   &imageAccess2,
						},
					},
				},
			}
			Expect(cdv2.DefaultComponent(cd)).To(Succeed())
			componentVersion := testutils.NewTestComponentVersionFromReader(cd)

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
			if blue.DeployExecutions[0].Type == "GoTemplate" {
				Expect(config).To(HaveKeyWithValue("images", []interface{}{
					map[string]interface{}{"image": "quay.io/example/myimage:1.0.0"},
					map[string]interface{}{"image": "quay.io/example/yourimage:1.0.0"},
				}))
			}
		},
			Entry("template with component descriptor v2", common.SCHEMA_VERSION_V2),
			Entry("template with component descriptor v3alpha1", common.SCHEMA_VERSION_V3ALPHA1),
		)

		DescribeTable("should use a resource from the component descriptor's referenced component", func(ocmSchemaVersion string) {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-10.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.Annotations = map[string]string{common.OCM_SCHEMA_VERSION: ocmSchemaVersion}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			imageAccess, err := testutils.NewOCIRegistryAccess("quay.io/example/myimage:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			cd := &types.ComponentDescriptor{
				Metadata: types.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/mycomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []*types.UnstructuredTypedObject{},
					Provider:           "internal",
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
			componentVersion := testutils.NewTestComponentVersionFromReader(cd)

			cd2 := types.ComponentDescriptor{
				Metadata: types.Metadata{Version: cdv2.SchemaVersion},
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "example.com/myrefcomp",
						Version: "1.0.0",
					},
					RepositoryContexts: []*types.UnstructuredTypedObject{},
					Provider:           "internal",
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
			componentVersion2 := testutils.NewTestComponentVersionFromReader(&cd2)

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
		},
			Entry("template with component descriptor v2", common.SCHEMA_VERSION_V2),
			Entry("template with component descriptor v3alpha1", common.SCHEMA_VERSION_V3ALPHA1),
		)

		DescribeTable("templating against specific component descriptor schema versions", func(useOCM bool, schemaVersion string, templateFileName string, schemaVersionSuffix string) {
			// Preparation to conveniently be able to access the respective component versions
			registry, err := testutils.NewLocalRegistryAccess(ctx, filepath.Join(sharedTestdataDir, "localocmrepository"))
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-" + schemaVersionSuffix, Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())
			componentVersionRef1, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-" + schemaVersionSuffix + "-ref1", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())
			componentVersionRef2, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-" + schemaVersionSuffix + "-ref2", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())

			componentVersionList := &model.ComponentVersionList{
				Components: []model.ComponentVersion{
					componentVersion,
					componentVersionRef1,
					componentVersionRef2,
				},
			}

			// Actual templating logic
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, templateFileName))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			// Templating schema version will be determined by this annotation
			blue.Annotations = map[string]string{common.OCM_SCHEMA_VERSION: schemaVersion}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(
				template.NewDeployExecutionOptions(
					template.NewBlueprintExecutionOptions(
						nil,
						&blueprints.Blueprint{Info: blue, Fs: nil},
						componentVersion,
						componentVersionList,
						nil)))

			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("name", "example.com/landscaper-component-"+schemaVersionSuffix))
			Expect(config).To(HaveKeyWithValue("names", []interface{}{
				map[string]interface{}{"name": "example.com/landscaper-component-" + schemaVersionSuffix},
				map[string]interface{}{"name": "example.com/landscaper-component-" + schemaVersionSuffix + "-ref1"},
				map[string]interface{}{"name": "example.com/landscaper-component-" + schemaVersionSuffix + "-ref2"},
			}))
		},
			Entry("default to schema version v2 with cnudie facade implementation", false, "", "template-30.yaml", "v2"),
			Entry("default to schema version v2 with ocmlib facade implementation", true, "", "template-30.yaml", "v2"),
			Entry("default to schema version v3alpha1 with ocmlib facade implementation", true, "", "template-31.yaml", "v3alpha1"),
			Entry("set schema version through blueprint to v2 - with cnudie facade implementation", false, common.SCHEMA_VERSION_V2, "template-30.yaml", "v2"),
			Entry("set schema version through blueprint to v2 - with ocmlib facade implementation", true, common.SCHEMA_VERSION_V2, "template-30.yaml", "v2"),
			Entry("set schema version through blueprint to v3alpha1 - with ocmlib facade implementation", true, common.SCHEMA_VERSION_V3ALPHA1, "template-31.yaml", "v3alpha1"),
		)

		It("templating against v2 with mixed component descriptor schema versions", func() {
			// Preparation to conveniently be able to access the respective component versions
			registry, err := testutils.NewLocalRegistryAccess(ctx, filepath.Join(sharedTestdataDir, "localocmrepository"))
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-v2-mixed", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())
			componentVersionRef1, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-v2-ref1", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())
			componentVersionRef2, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-v3alpha1-ref2", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())

			componentVersionList := &model.ComponentVersionList{
				Components: []model.ComponentVersion{
					componentVersion,
					componentVersionRef1,
					componentVersionRef2,
				},
			}

			// Actual templating logic
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-30.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			// Templating schema version will be determined by this annotation
			// blue.Annotations = map[string]string{common.OCM_SCHEMA_VERSION: common.SCHEMA_VERSION_V2}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(
				template.NewDeployExecutionOptions(
					template.NewBlueprintExecutionOptions(
						nil,
						&blueprints.Blueprint{Info: blue, Fs: nil},
						componentVersion,
						componentVersionList,
						nil)))

			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("name", "example.com/landscaper-component-v2-mixed"))
			Expect(config).To(HaveKeyWithValue("names", []interface{}{
				map[string]interface{}{"name": "example.com/landscaper-component-v2-mixed"},
				map[string]interface{}{"name": "example.com/landscaper-component-v2-ref1"},
				map[string]interface{}{"name": "example.com/landscaper-component-v3alpha1-ref2"},
			}))
		})

		It("templating against v3alpha1 with mixed component descriptor schema versions", func() {
			// Preparation to conveniently be able to access the respective component versions
			registry, err := testutils.NewLocalRegistryAccess(ctx, filepath.Join(sharedTestdataDir, "localocmrepository"))
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-v3alpha1-mixed", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())
			componentVersionRef1, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-v2-ref1", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())
			componentVersionRef2, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "example.com/landscaper-component-v3alpha1-ref2", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())

			componentVersionList := &model.ComponentVersionList{
				Components: []model.ComponentVersion{
					componentVersion,
					componentVersionRef1,
					componentVersionRef2,
				},
			}

			// Actual templating logic
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-31.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			// Templating schema version will be determined by this annotation
			// blue.Annotations = map[string]string{common.OCM_SCHEMA_VERSION: common.SCHEMA_VERSION_V2}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(
				template.NewDeployExecutionOptions(
					template.NewBlueprintExecutionOptions(
						nil,
						&blueprints.Blueprint{Info: blue, Fs: nil},
						componentVersion,
						componentVersionList,
						nil)))

			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("name", "example.com/landscaper-component-v3alpha1-mixed"))
			Expect(config).To(HaveKeyWithValue("names", []interface{}{
				map[string]interface{}{"name": "example.com/landscaper-component-v3alpha1-mixed"},
				map[string]interface{}{"name": "example.com/landscaper-component-v2-ref1"},
				map[string]interface{}{"name": "example.com/landscaper-component-v3alpha1-ref2"},
			}))
		})

		It("should get resource key for given relative resource reference", func() {
			// Preparation to conveniently be able to access the respective component versions
			repositoryContext := &cdv2.UnstructuredTypedObject{}
			Expect(repositoryContext.UnmarshalJSON([]byte(`{"type": "CommonTransportFormat/v1","filePath": "testdata/shared_data/ctf-local-blobs", "fileFormat": "directory"}`))).To(Succeed())
			registry, err := registries.GetFactory(true).NewRegistryAccess(ctx, &model.RegistryAccessOptions{
				LocalRegistryConfig: &apiconfig.LocalRegistryConfiguration{RootPath: filepath.Join(sharedTestdataDir, "ctf-local-blobs")},
				AdditionalRepositoryContexts: []types.PrioritizedRepositoryContext{
					{
						RepositoryContext: repositoryContext,
						Priority:          10,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "github.com/root", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())

			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-32.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(
					nil,
					&blueprints.Blueprint{Info: blue, Fs: nil},
					componentVersion,
					nil,
					nil,
				),
			))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeNil())
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config["key"]).ToNot(BeEmpty())
		})

		It("should get resource key for given relative resource path", func() {
			// Preparation to conveniently be able to access the respective component versions
			repositoryContext := &cdv2.UnstructuredTypedObject{}
			Expect(repositoryContext.UnmarshalJSON([]byte(`{"type": "CommonTransportFormat/v1","filePath": "testdata/shared_data/ctf-local-blobs", "fileFormat": "directory"}`))).To(Succeed())
			registry, err := registries.GetFactory(true).NewRegistryAccess(ctx, &model.RegistryAccessOptions{
				LocalRegistryConfig: &apiconfig.LocalRegistryConfiguration{RootPath: filepath.Join(sharedTestdataDir, "ctf-local-blobs")},
				AdditionalRepositoryContexts: []types.PrioritizedRepositoryContext{
					{
						RepositoryContext: repositoryContext,
						Priority:          10,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "github.com/root", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())

			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-34.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(
					nil,
					&blueprints.Blueprint{Info: blue, Fs: nil},
					componentVersion,
					nil,
					nil,
				),
			))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeNil())
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config["key"]).ToNot(BeEmpty())
		})

		It("should get resource content for given relative resource reference", func() {
			// Preparation to conveniently be able to access the respective component versions
			repositoryContext := &cdv2.UnstructuredTypedObject{}
			Expect(repositoryContext.UnmarshalJSON([]byte(`{"type": "CommonTransportFormat/v1","filePath": "testdata/shared_data/ctf-local-blobs", "fileFormat": "directory"}`))).To(Succeed())
			registry, err := registries.GetFactory(true).NewRegistryAccess(ctx, &model.RegistryAccessOptions{
				LocalRegistryConfig: &apiconfig.LocalRegistryConfiguration{RootPath: filepath.Join(sharedTestdataDir, "ctf-local-blobs")},
				AdditionalRepositoryContexts: []types.PrioritizedRepositoryContext{
					{
						RepositoryContext: repositoryContext,
						Priority:          10,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "github.com/root", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())

			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-33.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(
					nil,
					&blueprints.Blueprint{Info: blue, Fs: nil},
					componentVersion,
					nil,
					nil,
				),
			))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeNil())
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config["content"]).ToNot(BeEmpty())
			data, err := runtime.DefaultYAMLEncoding.Marshal(res)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).ToNot(BeNil())
		})

		It("should get resource content for given relative resource path", func() {
			// Preparation to conveniently be able to access the respective component versions
			repositoryContext := &cdv2.UnstructuredTypedObject{}
			Expect(repositoryContext.UnmarshalJSON([]byte(`{"type": "CommonTransportFormat/v1","filePath": "testdata/shared_data/ctf-local-blobs", "fileFormat": "directory"}`))).To(Succeed())
			registry, err := registries.GetFactory(true).NewRegistryAccess(ctx, &model.RegistryAccessOptions{
				LocalRegistryConfig: &apiconfig.LocalRegistryConfiguration{RootPath: filepath.Join(sharedTestdataDir, "ctf-local-blobs")},
				AdditionalRepositoryContexts: []types.PrioritizedRepositoryContext{
					{
						RepositoryContext: repositoryContext,
						Priority:          10,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registry.GetComponentVersion(ctx, &types.ComponentVersionKey{Name: "github.com/root", Version: "1.0.0"})
			Expect(err).ToNot(HaveOccurred())

			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-35.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(
					nil,
					&blueprints.Blueprint{Info: blue, Fs: nil},
					componentVersion,
					nil,
					nil,
				),
			))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeNil())
			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config["content"]).ToNot(BeEmpty())
			data, err := runtime.DefaultYAMLEncoding.Marshal(res)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).ToNot(BeNil())
		})

		It("should throw an error when the template tries to template a undefined value", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-08.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			_, err = op.TemplateDeployExecutions(template.NewDeployExecutionOptions(
				template.NewBlueprintExecutionOptions(nil, &blueprints.Blueprint{Info: blue, Fs: nil}, nil, nil,
					map[string]interface{}{"version": "0.0.1"})))
			Expect(err).ToNot(HaveOccurred())

			state := map[string]string{
				"version": "0.0.2",
			}
			stateBytes, err := json.Marshal(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(stateHandler.Store(ctx, "deploy"+exec[0].Name, stateBytes)).To(Succeed())
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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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

		It("should use a parsed oci ref to template", func() {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, "template-13.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

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

func runTestSuiteGoTemplate(testdataDir string) {
	var (
		stateHandler template.GenericStateHandler

		executeTemplate = func(templateFile string, imports map[string]interface{}) ([]template.DeployItemSpecification, error) {
			tmpl, err := os.ReadFile(filepath.Join(testdataDir, templateFile))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec

			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(
				template.NewDeployExecutionOptions(
					template.NewBlueprintExecutionOptions(
						nil,
						&blueprints.Blueprint{Info: blue, Fs: nil},
						nil,
						nil,
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

func runTestSuiteSpiff(testdataDir string) {
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
			op := template.New(gotemplate.New(stateHandler, nil), spiff.New(stateHandler, nil))

			res, err := op.TemplateDeployExecutions(
				template.NewDeployExecutionOptions(
					template.NewBlueprintExecutionOptions(
						nil,
						&blueprints.Blueprint{Info: blue, Fs: nil},
						nil,
						nil,
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
