package landscaper_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
)

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

var _ = Describe("Landscaper", func() {

	Context("Render blueprint", func() {

		It("should render a blueprint that imports a target list", func() {
			renderer := lsutils.NewBlueprintRenderer(nil, nil, nil)
			_, err := renderer.RenderDeployItemsAndSubInstallations(&lsutils.ResolvedInstallation{
				ComponentVersion: nil,
				Installation:     nil,
				Blueprint:        GetBlueprint("./testdata/00-blueprint-with-targetlist/blueprint"),
			}, GetImports("./testdata/00-blueprint-with-targetlist/values.yaml"))

			Expect(err).ToNot(HaveOccurred())
		})

	})

})
