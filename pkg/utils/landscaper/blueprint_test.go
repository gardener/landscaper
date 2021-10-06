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

		It("should render a blueprint with a target list import", func() {
			blueprintRenderArgs := BlueprintRenderArgs{
				Fs:                   osfs.New(),
				ImportValuesFilepath: "./testdata/00-blueprint-with-targetlist/values.yaml",
				RootDir:              "./testdata/00-blueprint-with-targetlist",
			}

			_, err := RenderBlueprint(blueprintRenderArgs)
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
