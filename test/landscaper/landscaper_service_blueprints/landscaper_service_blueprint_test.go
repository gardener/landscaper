package landscaper_service_blueprints_test

import (
	"context"
	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/pkg/deployer/helm"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Service Blueprint Test Suite")
}

type Empty struct {

}

const projectRoot = "../../../"
const testData = "testdata"

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
		registry componentsregistry.TypedRegistry
		repository *componentsregistry.LocalRepository
	)

	BeforeSuite(func() {
		var err error
		registry, err = componentsregistry.NewLocalClient(logr.Discard(), "./testdata/registry")
		Expect(err).ToNot(HaveOccurred())
		repository = componentsregistry.NewLocalRepository("./testdata/registry")

		err = copyFile(filepath.Join(projectRoot, ".landscaper/landscaper-service/definition/landscaper-configuration.json"), filepath.Join(testData, "registry/landscaper-service/blobs/landscaper-configuration/schema.json"))
		Expect(err).ToNot(HaveOccurred())
		err = copyFile(filepath.Join(projectRoot, ".landscaper/landscaper-service/definition/registry-configuration.json"), filepath.Join(testData, "registry/landscaper-service/blobs/registry-configuration/schema.json"))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterSuite(func() {
		os.Remove(filepath.Join(testData, "registry/landscaper-service/blobs/landscaper-configuration/schema.json"))
		os.Remove(filepath.Join(testData, "registry/landscaper-service/blobs/registry-configuration/schema.json"))
	})

	It("landscaper", func() {
		landscaperServiceCD, err := registry.Resolve(context.Background(), repository, "github.com/gardener/landscaper/landscaper-service", "v0.20.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(landscaperServiceCD).ToNot(BeNil())

		landscaperCD, err := registry.Resolve(context.Background(), repository, "github.com/gardener/landscaper", "v0.20.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(landscaperCD).ToNot(BeNil())

		cdList := &v2.ComponentDescriptorList{
			Components: []v2.ComponentDescriptor{
				*landscaperCD,
			},
		}

		out, err := lsutils.RenderBlueprint(lsutils.BlueprintRenderArgs{
			Fs:      osfs.New(),
			RootDir: filepath.Join(projectRoot, ".landscaper/landscaper-service"),
			ComponentDescriptor: landscaperServiceCD,
			ComponentDescriptorList: cdList,
			ComponentResolver: registry,
			BlueprintPath: filepath.Join(projectRoot, ".landscaper/landscaper-service/blueprint/landscaper"),
			ImportValuesFilepath: filepath.Join(testData, "imports.yaml"),
		})
		testutils.ExpectNoError(err)
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
	})
})
