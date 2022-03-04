// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package landscaper_service_blueprints_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
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
	testData = "testdata"
)

var filesToCopy = map[string]string{
	filepath.Join(projectRoot, ".landscaper/landscaper-service/definition/landscaper-configuration.json"): filepath.Join(testData, "registry/landscaper-service/blobs/landscaper-configuration/schema.json"),
	filepath.Join(projectRoot, ".landscaper/landscaper-service/definition/registry-configuration.json"): filepath.Join(testData, "registry/landscaper-service/blobs/registry-configuration/schema.json"),
}

func copyFile(source, dest string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

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

var _ = Describe("Blueprint", func() {

	var (
		registry   componentsregistry.TypedRegistry
		repository *componentsregistry.LocalRepository
		landscaperServiceCD *cdv2.ComponentDescriptor
		landscaperCD *cdv2.ComponentDescriptor
		virtualGardenCD *cdv2.ComponentDescriptor
		cdList cdv2.ComponentDescriptorList
		repositoryContext cdv2.UnstructuredTypedObject
	)

	BeforeSuite(func() {
		var err error
		registry, err = componentsregistry.NewLocalClient(logr.Discard(), "./testdata/registry")
		Expect(err).ToNot(HaveOccurred())
		repository = componentsregistry.NewLocalRepository("./testdata/registry")

		for source, dest := range filesToCopy {
			err = copyFile(source, dest)
			Expect(err).ToNot(HaveOccurred())
		}

		landscaperServiceCD, err = registry.Resolve(context.Background(), repository, "github.com/gardener/landscaper/landscaper-service", "v0.20.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(landscaperServiceCD).ToNot(BeNil())

		landscaperCD, err = registry.Resolve(context.Background(), repository, "github.com/gardener/landscaper", "v0.20.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(landscaperCD).ToNot(BeNil())

		virtualGardenCD, err = registry.Resolve(context.Background(), repository, "github.com/gardener/virtual-garden", "v0.1.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(landscaperCD).ToNot(BeNil())

		cdList.Components = []cdv2.ComponentDescriptor{
			*landscaperCD,
			*virtualGardenCD,
		}

		repoCtx := &cdv2.OCIRegistryRepository{
			ObjectType: cdv2.ObjectType{
				Type: registry.Type(),
			},
			BaseURL:  filepath.Join(testData, "registry"),
		}

		repositoryContext, err = cdv2.NewUnstructured(repoCtx)
		Expect(err).ToNot(HaveOccurred())

	})

	AfterSuite(func() {
		for _, dest := range filesToCopy {
			os.Remove(dest)
		}
	})

	It("landscaper", func() {
		out, err := lsutils.RenderBlueprint(lsutils.BlueprintRenderArgs{
			Fs:                      osfs.New(),
			RootDir:                 filepath.Join(projectRoot, ".landscaper/landscaper-service"),
			ComponentDescriptor:     landscaperServiceCD,
			ComponentDescriptorList: &cdList,
			ComponentResolver:       registry,
			BlueprintPath:           filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/landscaper"),
			ImportValuesFilepath:    filepath.Join(testData, "imports-landscaper.yaml"),
			RepositoryContext: &repositoryContext,
		})
		testutils.ExpectNoError(err)
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
	})

	It("rbac", func() {
		out, err := lsutils.RenderBlueprint(lsutils.BlueprintRenderArgs{
			Fs:                      osfs.New(),
			RootDir:                 filepath.Join(projectRoot, ".landscaper/landscaper-service"),
			ComponentDescriptor:     landscaperServiceCD,
			ComponentDescriptorList: &cdList,
			ComponentResolver:       registry,
			BlueprintPath:           filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/rbac"),
			ImportValuesFilepath:    filepath.Join(testData, "imports-rbac.yaml"),
		})
		testutils.ExpectNoError(err)
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
	})

	It("installation", func() {
		out, err := lsutils.RenderBlueprint(lsutils.BlueprintRenderArgs{
			Fs:                      osfs.New(),
			RootDir:                 filepath.Join(projectRoot, ".landscaper/landscaper-service"),
			ComponentDescriptor:     landscaperServiceCD,
			ComponentDescriptorList: &cdList,
			ComponentResolver:       registry,
			BlueprintPath:           filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/installation"),
			ImportValuesFilepath:    filepath.Join(testData, "imports-installation.yaml"),
		})
		testutils.ExpectNoError(err)
		Expect(out.Installations).To(HaveLen(3))
	})
})
