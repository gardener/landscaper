package landscaper

import (
	"testing"

	"github.com/mandelsoft/vfs/pkg/osfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper utils Test Suite")
}

var _ = Describe("Landscaper", func() {

	Context("Render blueprint", func() {

		It("should render a blueprint that imports a target list", func() {
			blueprintRenderArgs := BlueprintRenderArgs{
				Fs:                   osfs.New(),
				ImportValuesFilepath: "./testdata/00-blueprint-with-targetlist/values.yaml",
				RootDir:              "./testdata/00-blueprint-with-targetlist",
			}

			_, err := RenderBlueprint(blueprintRenderArgs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should render a blueprint that imports a component descriptor and a component descriptor list", func() {
			blueprintRenderArgs := BlueprintRenderArgs{
				Fs:                   osfs.New(),
				ImportValuesFilepath: "./testdata/01-blueprint-with-cdlist/values.yaml",
				RootDir:              "./testdata/01-blueprint-with-cdlist",
			}

			_, err := RenderBlueprint(blueprintRenderArgs)
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
