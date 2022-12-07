// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package landscaper_service_blueprints_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/osfs"

	"github.com/gardener/landscaper/pkg/deployer/helm"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	testutils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Service Blueprint Test Suite")
}

const (
	projectRoot = "../../../"
	testData    = "testdata"
)

var filesToCopy = map[string]string{
	filepath.Join(projectRoot, ".landscaper/landscaper-service/definition/landscaper-configuration.json"):   filepath.Join(testData, "registry/landscaper-service/blobs/landscaper-configuration/schema.json"),
	filepath.Join(projectRoot, ".landscaper/landscaper-service/definition/registry-configuration.json"):     filepath.Join(testData, "registry/landscaper-service/blobs/registry-configuration/schema.json"),
	filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/landscaper/blueprint.yaml"):        filepath.Join(testData, "registry/landscaper-service/blobs/landscaper-blueprint/blueprint.yaml"),
	filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/landscaper/deploy-execution.yaml"): filepath.Join(testData, "registry/landscaper-service/blobs/landscaper-blueprint/deploy-execution.yaml"),
	filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/rbac/blueprint.yaml"):              filepath.Join(testData, "registry/landscaper-service/blobs/rbac-blueprint/blueprint.yaml"),
	filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/rbac/deploy-execution.yaml"):       filepath.Join(testData, "registry/landscaper-service/blobs/rbac-blueprint/deploy-execution.yaml"),
	filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/rbac/export-execution.yaml"):       filepath.Join(testData, "registry/landscaper-service/blobs/rbac-blueprint/export-execution.yaml"),
}

func copyFile(source, dest string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	err = os.MkdirAll(filepath.Dir(dest), os.ModePerm)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func GetBlueprint(path string) *blueprints.Blueprint {
	fs := osfs.New()
	blueprintsFs, err := projectionfs.New(fs, path)
	Expect(err).ToNot(HaveOccurred())
	blueprint, err := blueprints.NewFromFs(blueprintsFs)
	Expect(err).ToNot(HaveOccurred())
	return blueprint
}

func GetImports(path string) map[string]interface{} {
	fs := osfs.New()
	importsRaw, err := vfs.ReadFile(fs, path)
	Expect(err).ToNot(HaveOccurred())
	var imports map[string]interface{}
	err = yaml.Unmarshal(importsRaw, &imports)
	Expect(err).ToNot(HaveOccurred())
	imports, ok := imports["imports"].(map[string]interface{})
	Expect(ok).To(BeTrue())
	return imports
}

var (
	registry            componentsregistry.TypedRegistry
	repository          *componentsregistry.LocalRepository
	landscaperServiceCD *cdv2.ComponentDescriptor
	landscaperCD        *cdv2.ComponentDescriptor
	virtualGardenCD     *cdv2.ComponentDescriptor
	cdList              cdv2.ComponentDescriptorList
	repositoryContext   cdv2.UnstructuredTypedObject
)

var _ = BeforeSuite(func() {
	var err error
	ctx := context.Background()
	defer ctx.Done()

	registry, err = componentsregistry.NewLocalClient("./testdata/registry")
	Expect(err).ToNot(HaveOccurred())
	repository = componentsregistry.NewLocalRepository("./testdata/registry")

	for source, dest := range filesToCopy {
		err = copyFile(source, dest)
		Expect(err).ToNot(HaveOccurred())
	}

	landscaperServiceCD, err = registry.Resolve(ctx, repository, "github.com/gardener/landscaper/landscaper-service", "v0.20.0")
	Expect(err).ToNot(HaveOccurred())
	Expect(landscaperServiceCD).ToNot(BeNil())

	landscaperCD, err = registry.Resolve(ctx, repository, "github.com/gardener/landscaper", "v0.20.0")
	Expect(err).ToNot(HaveOccurred())
	Expect(landscaperCD).ToNot(BeNil())

	virtualGardenCD, err = registry.Resolve(ctx, repository, "github.com/gardener/virtual-garden", "v0.1.0")
	Expect(err).ToNot(HaveOccurred())
	Expect(landscaperCD).ToNot(BeNil())

	cdList.Components = []cdv2.ComponentDescriptor{
		*landscaperServiceCD,
		*landscaperCD,
		*virtualGardenCD,
	}

	repoCtx := &cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: registry.Type(),
		},
		BaseURL: filepath.Join(testData, "registry"),
	}

	repositoryContext.ObjectType = repoCtx.ObjectType
	repositoryContext.Raw, err = json.Marshal(repoCtx)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	for _, dest := range filesToCopy {
		_ = os.WriteFile(dest, []byte("{}"), 0644)
	}
})

var _ = Describe("Landscaper Service Component", func() {

	It("should install the landscaper blueprint", func() {
		renderer := lsutils.NewBlueprintRenderer(&cdList, registry, &repositoryContext)
		out, err := renderer.RenderDeployItemsAndSubInstallations(&lsutils.ResolvedInstallation{
			ComponentDescriptor: landscaperServiceCD,
			Installation:        &lsv1alpha1.Installation{},
			Blueprint:           GetBlueprint(filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/landscaper")),
		}, GetImports(filepath.Join(testData, "imports-landscaper.yaml")))

		testutils.ExpectNoError(err)
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
	})

	It("should install the rbac blueprint", func() {
		renderer := lsutils.NewBlueprintRenderer(&cdList, registry, &repositoryContext)
		out, err := renderer.RenderDeployItemsAndSubInstallations(&lsutils.ResolvedInstallation{
			ComponentDescriptor: landscaperServiceCD,
			Installation:        &lsv1alpha1.Installation{},
			Blueprint:           GetBlueprint(filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/rbac")),
		}, GetImports(filepath.Join(testData, "imports-rbac.yaml")))

		testutils.ExpectNoError(err)
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
	})

	It("should install the installation blueprint", func() {
		renderer := lsutils.NewBlueprintRenderer(&cdList, registry, &repositoryContext)
		out, err := renderer.RenderDeployItemsAndSubInstallations(&lsutils.ResolvedInstallation{
			ComponentDescriptor: landscaperServiceCD,
			Installation:        &lsv1alpha1.Installation{},
			Blueprint:           GetBlueprint(filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/installation")),
		}, GetImports(filepath.Join(testData, "imports-installation.yaml")))

		testutils.ExpectNoError(err)
		Expect(out.Installations).To(HaveLen(3))
	})
})
