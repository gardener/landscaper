// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package landscaper_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/gardener/landscaper/apis/config"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
)

type TestSimulatorCallbacks struct {
	installations     map[string]*lsv1alpha1.Installation
	installationState map[string]map[string][]byte
	deployItems       map[string]*lsv1alpha1.DeployItem
	deployItemsState  map[string]map[string][]byte
	imports           map[string]interface{}
	exports           map[string]interface{}
}

func (c *TestSimulatorCallbacks) OnInstallation(path string, installation *lsv1alpha1.Installation) {
	c.installations[path] = installation
}

func (c *TestSimulatorCallbacks) OnInstallationTemplateState(path string, state map[string][]byte) {
	c.installationState[path] = state
}

func (c *TestSimulatorCallbacks) OnImports(path string, imports map[string]interface{}) {
	c.imports[path] = imports
}

func (c *TestSimulatorCallbacks) OnDeployItem(path string, deployItem *lsv1alpha1.DeployItem) {
	c.deployItems[fmt.Sprintf("%s/%s", path, deployItem.Name)] = deployItem
}

func (c *TestSimulatorCallbacks) OnDeployItemTemplateState(path string, state map[string][]byte) {
	c.deployItemsState[path] = state
}

func (c *TestSimulatorCallbacks) OnExports(path string, exports map[string]interface{}) {
	c.exports[path] = exports
}

var _ = Describe("Installation Simulator", func() {
	var (
		testDataDir          = "./testdata/01-subinstallations"
		registryAccess       model.RegistryAccess
		rootComponentVersion model.ComponentVersion
		componentVersionList *model.ComponentVersionList
		blueprint            *blueprints.Blueprint
		repositoryContext    types.UnstructuredTypedObject
		exportTemplates      lsutils.ExportTemplates
		callbacks            = &TestSimulatorCallbacks{
			installations:     make(map[string]*lsv1alpha1.Installation),
			installationState: make(map[string]map[string][]byte),
			deployItems:       make(map[string]*lsv1alpha1.DeployItem),
			deployItemsState:  make(map[string]map[string][]byte),
			imports:           make(map[string]interface{}),
			exports:           make(map[string]interface{}),
		}
	)

	BeforeEach(func() {
		var err error
		ctx := context.Background()
		defer ctx.Done()

		localregistryconfig := &config.LocalRegistryConfiguration{RootPath: testDataDir}
		registryAccess, err = registries.GetFactory().NewRegistryAccess(ctx, nil, nil, localregistryconfig, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(repositoryContext.UnmarshalJSON([]byte(`{"type":"local"}`))).To(Succeed())

		rootComponentVersion, err = registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &repositoryContext,
			ComponentName:     "example.com/root",
			Version:           "v0.1.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(rootComponentVersion).ToNot(BeNil())

		componentVersionA, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &repositoryContext,
			ComponentName:     "example.com/componenta",
			Version:           "v0.1.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(componentVersionA).ToNot(BeNil())

		componentVersionB, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &repositoryContext,
			ComponentName:     "example.com/componentb",
			Version:           "v0.1.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(componentVersionA).ToNot(BeNil())

		componentVersionList = &model.ComponentVersionList{
			Components: []model.ComponentVersion{
				rootComponentVersion,
				componentVersionA,
				componentVersionB,
			},
		}

		fs := osfs.New()
		blueprintsFs, err := projectionfs.New(fs, path.Join(testDataDir, "blobs/root-blueprint"))
		Expect(err).ToNot(HaveOccurred())

		blueprint, err = blueprints.NewFromFs(blueprintsFs)
		Expect(err).ToNot(HaveOccurred())

		repoCtx := &cdv2.OCIRegistryRepository{
			ObjectType: cdv2.ObjectType{
				Type: repositoryContext.GetType(),
			},
			BaseURL: testDataDir,
		}

		repositoryContext.ObjectType = repoCtx.ObjectType
		repositoryContext.Raw, err = json.Marshal(repoCtx)
		Expect(err).ToNot(HaveOccurred())

		exportTemplates.DeployItemExports = []*lsutils.ExportTemplate{
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

		exportTemplates.InstallationExports = []*lsutils.ExportTemplate{
			{
				Name:     "subinst-c",
				Selector: ".*/subinst-c",
				Template: `
dataExports:
  subinst-c-export: {{ .installation.metadata.name }}
targetExports: []
`,
			},
		}
	})

	It("should simulate an installation with subinstallations", func() {
		simulator, err := lsutils.NewInstallationSimulator(componentVersionList, registryAccess, &repositoryContext, exportTemplates)
		Expect(err).ToNot(HaveOccurred())
		simulator.SetCallbacks(callbacks)

		cluster := lsv1alpha1.Target{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster",
				Namespace: "default",
			},
			Spec: lsv1alpha1.TargetSpec{
				Type:          targettypes.KubernetesClusterTargetType,
				Configuration: lsv1alpha1.NewAnyJSONPointer([]byte("{ \"kubeconfig\": \"{}\" }")),
			},
		}

		clusterList := []lsv1alpha1.Target{
			cluster,
		}

		marshaled, err := yaml.Marshal(cluster)
		Expect(err).ToNot(HaveOccurred())
		var clusterMap map[string]interface{}
		err = yaml.Unmarshal(marshaled, &clusterMap)
		Expect(err).ToNot(HaveOccurred())

		marshaled, err = yaml.Marshal(clusterList)
		Expect(err).ToNot(HaveOccurred())
		var clusterListMap []interface{}
		err = yaml.Unmarshal(marshaled, &clusterListMap)
		Expect(err).ToNot(HaveOccurred())

		imports := map[string]interface{}{
			"root-param-a": "valua-a",
			"root-param-b": "value-b",
			"cluster":      clusterMap,
			"clusters":     clusterListMap,
		}

		exports, err := simulator.Run(rootComponentVersion, blueprint, imports)
		Expect(err).ToNot(HaveOccurred())
		Expect(exports).ToNot(BeNil())

		Expect(exports.DataObjects).To(HaveLen(3))
		Expect(exports.DataObjects).To(HaveKey("export-root-a"))
		Expect(exports.DataObjects).To(HaveKey("export-root-b"))
		Expect(exports.DataObjects).To(HaveKey("export-root-c"))
		Expect(exports.DataObjects["export-root-a"]).To(Equal("subinst-a-deploy"))
		Expect(exports.DataObjects["export-root-b"]).To(Equal("example.com/componentb"))
		Expect(exports.DataObjects["export-root-c"]).To(Equal("subinst-c"))

		Expect(exports.Targets).To(HaveLen(1))
		Expect(exports.Targets).To(HaveKey("export-root-target"))
		marshalledTarget, err := json.Marshal(exports.Targets["export-root-target"])
		Expect(err).ToNot(HaveOccurred())
		target := &lsv1alpha1.Target{}
		err = json.Unmarshal(marshalledTarget, target)
		Expect(err).ToNot(HaveOccurred())
		Expect(target.Spec.Type).To(Equal(targettypes.KubernetesClusterTargetType))
		Expect(target.Spec.Configuration).ToNot(BeNil())

		Expect(callbacks.installations).To(HaveLen(4))
		Expect(callbacks.installations).To(HaveKey("root"))
		Expect(callbacks.installations).To(HaveKey("root/subinst-a"))
		Expect(callbacks.installations).To(HaveKey("root/subinst-b"))
		Expect(callbacks.installations).To(HaveKey("root/subinst-c"))
		Expect(callbacks.installations["root/subinst-a"].Name).To(Equal("subinst-a"))
		Expect(callbacks.installations["root/subinst-b"].Name).To(Equal("subinst-b"))
		Expect(callbacks.installations["root/subinst-c"].Name).To(Equal("subinst-c"))

		Expect(callbacks.deployItems).To(HaveLen(2))
		Expect(callbacks.deployItems).To(HaveKey("root/subinst-a/subinst-a-deploy"))
		Expect(callbacks.deployItems).To(HaveKey("root/subinst-b/subinst-b-deploy"))
		Expect(callbacks.deployItems["root/subinst-a/subinst-a-deploy"].Name).To(Equal("subinst-a-deploy"))
		Expect(callbacks.deployItems["root/subinst-b/subinst-b-deploy"].Name).To(Equal("subinst-b-deploy"))

		Expect(callbacks.imports).To(HaveLen(4))
		Expect(callbacks.imports).To(HaveKey("root"))
		Expect(callbacks.imports).To(HaveKey("root/subinst-a"))
		Expect(callbacks.imports).To(HaveKey("root/subinst-b"))
		Expect(callbacks.imports).To(HaveKey("root/subinst-c"))

		Expect(callbacks.imports["root"]).To(HaveKeyWithValue("foo", "bar"))

		Expect(callbacks.imports["root/subinst-a"]).To(HaveKey("subinst-a-param-a"))
		Expect(callbacks.imports["root/subinst-a"]).To(HaveKey("subinst-a-param-b"))
		Expect(callbacks.imports["root/subinst-a"]).To(HaveKey("cluster"))

		Expect(callbacks.imports["root/subinst-b"]).To(HaveKey("subinst-b-param-a"))
		Expect(callbacks.imports["root/subinst-b"]).To(HaveKey("subinst-b-param-b"))
		Expect(callbacks.imports["root/subinst-b"]).To(HaveKey("cluster"))
		Expect(callbacks.imports["root/subinst-b"]).ToNot(HaveKey("subinst-a-param-a"))
		Expect(callbacks.imports["root/subinst-b"]).ToNot(HaveKey("subinst-a-param-b"))

		Expect(callbacks.imports["root/subinst-b"].(map[string]interface{})["subinst-b-param-b"]).To(Equal("example.com/componenta"))

		Expect(callbacks.imports["root/subinst-c"]).To(HaveKey("clusters-a"))
		clustersImport, ok := callbacks.imports["root/subinst-c"].(map[string]interface{})["clusters-a"].([]interface{})
		Expect(ok).To(BeTrue())
		Expect(clustersImport).To(HaveLen(1))

		Expect(callbacks.imports["root/subinst-c"]).To(HaveKey("clusters-b"))
		clustersImport, ok = callbacks.imports["root/subinst-c"].(map[string]interface{})["clusters-b"].([]interface{})
		Expect(ok).To(BeTrue())
		Expect(clustersImport).To(HaveLen(1))

		Expect(callbacks.exports).To(HaveLen(4))
		Expect(callbacks.exports).To(HaveKey("root"))
		Expect(callbacks.exports).To(HaveKey("root/subinst-a"))
		Expect(callbacks.exports).To(HaveKey("root/subinst-b"))
		Expect(callbacks.exports).To(HaveKey("root/subinst-c"))

		Expect(callbacks.exports["root/subinst-a"]).To(HaveKey("subinst-a-export-a"))
		Expect(callbacks.exports["root/subinst-a"]).To(HaveKey("subinst-a-export-b"))

		Expect(callbacks.exports["root/subinst-b"]).To(HaveKey("subinst-b-export-a"))
		Expect(callbacks.exports["root/subinst-b"]).To(HaveKey("subinst-b-export-b"))
		Expect(callbacks.exports["root/subinst-b"]).ToNot(HaveKey("subinst-a-export-a"))
		Expect(callbacks.exports["root/subinst-b"]).ToNot(HaveKey("subinst-a-export-b"))
		Expect(callbacks.exports["root/subinst-b"]).ToNot(HaveKey("subinst-a-export-target"))

		Expect(callbacks.exports["root/subinst-c"]).To(HaveKey("subinst-c-export"))

		Expect(callbacks.deployItemsState).To(HaveLen(1))
		Expect(callbacks.deployItemsState).To(HaveKey("root/subinst-a"))
		Expect(callbacks.deployItemsState["root/subinst-a"]).To(HaveKey("deploydeploy-execution"))
		Expect(callbacks.deployItemsState["root/subinst-a"]["deploydeploy-execution"]).To(ContainSubstring("stateval"))
	})
})
