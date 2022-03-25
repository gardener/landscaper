// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package landscaper_test

import (
	"context"
	"encoding/json"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
)

var _ = Describe("Installation Simulator", func() {
	var (
		testDataDir       = "./testdata/02-subinstallations"
		registry          componentsregistry.TypedRegistry
		repository        *componentsregistry.LocalRepository
		cd                *cdv2.ComponentDescriptor
		cdList            cdv2.ComponentDescriptorList
		repositoryContext cdv2.UnstructuredTypedObject
		exportTemplates   lsutils.ExportTemplates
	)

	BeforeEach(func() {
		var err error
		ctx := context.Background()
		defer ctx.Done()

		registry, err = componentsregistry.NewLocalClient(logr.Discard(), testDataDir)
		Expect(err).ToNot(HaveOccurred())
		repository = componentsregistry.NewLocalRepository(testDataDir)

		root, err := registry.Resolve(ctx, repository, "example.com/root", "v0.1.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(root).ToNot(BeNil())

		componentA, err := registry.Resolve(ctx, repository, "example.com/componenta", "v0.1.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(componentA).ToNot(BeNil())

		componentB, err := registry.Resolve(ctx, repository, "example.com/componentb", "v0.1.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(componentB).ToNot(BeNil())

		cd = root
		cdList.Components = []cdv2.ComponentDescriptor{
			*root,
			*componentA,
			*componentB,
		}

		repoCtx := &cdv2.OCIRegistryRepository{
			ObjectType: cdv2.ObjectType{
				Type: registry.Type(),
			},
			BaseURL: testDataDir,
		}

		repositoryContext.ObjectType = repoCtx.ObjectType
		repositoryContext.Raw, err = json.Marshal(repoCtx)
		Expect(err).ToNot(HaveOccurred())

		exportTemplates.DeployItemExports = []*lsutils.DeployItemExportTemplate{
			{
				Name:     "subinst-a-deploy",
				Selector: ".*/subinst-a-deploy",
				Template: `
exports:
  subinst-a-export-a: {{ .deployItem.metadata.name }}
  subinst-a-export-b: {{ .cd.component.name }}
`,
				SelectorRegexp: nil,
			},
			{
				Name:     "subinst-b-deploy",
				Selector: ".*/subinst-b-deploy",
				Template: `
exports:
  subinst-b-export-a: {{ .deployItem.metadata.name }}
  subinst-b-export-b: {{ .cd.component.name }}
`,
				SelectorRegexp: nil,
			},
		}
	})

	It("should simulate an installation with subinstallations", func() {
		simulator, err := lsutils.NewInstallationSimulator(&cdList, registry, &repositoryContext, exportTemplates)
		Expect(err).ToNot(HaveOccurred())

		fs := osfs.New()
		blueprintsFs, err := projectionfs.New(fs, path.Join(testDataDir, "root/blobs/blueprint"))
		Expect(err).ToNot(HaveOccurred())

		blue, err := blueprints.NewFromFs(blueprintsFs)
		Expect(err).ToNot(HaveOccurred())

		cluster := lsv1alpha1.Target{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster",
				Namespace: "default",
			},
			Spec: lsv1alpha1.TargetSpec{
				Type:          lsv1alpha1.KubernetesClusterTargetType,
				Configuration: lsv1alpha1.NewAnyJSON([]byte("{ \"kubeconfig\": \"{}\" }")),
			},
		}

		marshaled, err := yaml.Marshal(cluster)
		Expect(err).ToNot(HaveOccurred())
		var clusterMap map[string]interface{}
		err = yaml.Unmarshal(marshaled, &clusterMap)
		Expect(err).ToNot(HaveOccurred())

		dataImports := map[string]interface{}{
			"root-param-a": "valua-a",
			"root-param-b": "value-b",
		}

		targetImports := map[string]interface{}{
			"cluster": clusterMap,
		}

		exports, err := simulator.Run(cd, blue, dataImports, targetImports)
		Expect(err).ToNot(HaveOccurred())
		Expect(exports).ToNot(BeNil())
		Expect(exports.DataObjects).To(HaveLen(2))
		Expect(exports.DataObjects).To(HaveKey("export-root-a"))
		Expect(exports.DataObjects).To(HaveKey("export-root-b"))
		Expect(exports.DataObjects["export-root-a"]).To(Equal("subinst-a-deploy"))
		Expect(exports.DataObjects["export-root-b"]).To(Equal("example.com/componentb"))
	})
})
