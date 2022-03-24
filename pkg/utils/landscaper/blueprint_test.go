package landscaper

import (
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"
	"testing"

	"github.com/mandelsoft/vfs/pkg/osfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper utils Test Suite")
}

func GetBlueprint(path string) *blueprints.Blueprint {
	fs := osfs.New()
	blueprintsFs, err := projectionfs.New(fs, path)
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

var _ = Describe("Landscaper", func() {

	Context("Render blueprint", func() {

		It("should render a blueprint that imports a target list", func() {
			renderer := NewBlueprintRenderer(nil, nil, nil)
			_, err := renderer.RenderDeployItemsAndSubInstallations(&RenderInput{
				ComponentDescriptor: nil,
				Installation:  nil,
				Blueprint: GetBlueprint("./testdata/00-blueprint-with-targetlist/blueprint"),
			}, GetImports("./testdata/00-blueprint-with-targetlist/values.yaml"))

			Expect(err).ToNot(HaveOccurred())
		})

		It("should render a blueprint that imports a component descriptor and a component descriptor list", func() {
			renderer := NewBlueprintRenderer(nil, nil, nil)
			_, err := renderer.RenderDeployItemsAndSubInstallations(&RenderInput{
				ComponentDescriptor: nil,
				Installation:  nil,
				Blueprint: GetBlueprint("./testdata/01-blueprint-with-cdlist/blueprint"),
			}, GetImports("./testdata/01-blueprint-with-cdlist/values.yaml"))

			Expect(err).ToNot(HaveOccurred())
		})

	})

})
