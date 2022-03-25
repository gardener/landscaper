// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployer_blueprints_test

import (
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"

	"github.com/mandelsoft/vfs/pkg/osfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/deployer/helm"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	testutils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Container Deployer Test Suite")
}

const projectRoot = "../../../"

func RenderBlueprint(deployer string) *lsutils.RenderedDeployItemsSubInstallations {
	fs := osfs.New()
	overlayFs, err := projectionfs.New(fs, filepath.Join(projectRoot, ".landscaper", deployer))
	Expect(err).ToNot(HaveOccurred())

	blueprintsFs, err := projectionfs.New(overlayFs, "blueprint")
	Expect(err).ToNot(HaveOccurred())
	blueprint, err := blueprints.NewFromFs(blueprintsFs)
	Expect(err).ToNot(HaveOccurred())

	exampleFs, err := projectionfs.New(overlayFs, "example")
	Expect(err).ToNot(HaveOccurred())

	importsData, err := vfs.ReadFile(exampleFs, "imports.yaml")
	Expect(err).ToNot(HaveOccurred())
	var imports map[string]interface{}
	err = yaml.Unmarshal(importsData, &imports)
	Expect(err).ToNot(HaveOccurred())

	cdData, err := vfs.ReadFile(exampleFs, "component-descriptor.yaml")
	Expect(err).ToNot(HaveOccurred())
	var cd cdv2.ComponentDescriptor
	err = yaml.Unmarshal(cdData, &cd)
	Expect(err).ToNot(HaveOccurred())

	renderer := lsutils.NewBlueprintRenderer(&cdv2.ComponentDescriptorList{}, nil, nil)

	out, err := renderer.RenderDeployItemsAndSubInstallations(&lsutils.ResolvedInstallation{
		ComponentDescriptor: &cd,
		Installation:        &lsv1alpha1.Installation{},
		Blueprint:           blueprint,
	}, imports["imports"].(map[string]interface{}))
	testutils.ExpectNoError(err)
	return out
}

var _ = Describe("Blueprint", func() {

	It("ContainerDeployer", func() {
		out := RenderBlueprint("container-deployer")
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
		expectedConfig := `
{
  "apiVersion": "helm.deployer.landscaper.gardener.cloud/v1alpha1",
  "chart": {
    "ref": "eu.gcr.io/gardener-project/landscaper/charts/container-deployer-controller:v0.5.3"
  },
  "helmDeployment": false,
  "kind": "ProviderConfiguration",
  "name": "landscaper-container-deployer",
  "namespace": "container-deployer",
  "updateStrategy": "update",
  "values": {
    "deployer": {
      "identity": "my-id",
      "initContainer": {
        "repository": "eu.gcr.io/gardener-project/landscaper/container-init-controller",
        "tag": "v0.5.3"
      },
      "namespace": "",
      "oci": {
        "allowPlainHttp": false,
        "secrets": {}
      },
      "targetSelector": [
        {
          "annotations": [
            {
              "key": "abc",
              "operator": "=",
              "value": "xyz"
            }
          ]
        }
      ],
      "waitContainer": {
        "repository": "eu.gcr.io/gardener-project/landscaper/container-wait-controller",
        "tag": "v0.5.3"
      }
    },
    "image": {
      "pullPolicy": "IfNotPresent",
      "repository": "eu.gcr.io/gardener-project/landscaper/container-deployer-controller",
      "tag": "v0.5.3"
    },
    "replicaCount": 1
  }
}
`
		Expect(di.Spec.Configuration.Raw).To(MatchJSON(expectedConfig))
	})

	It("HelmDeployer", func() {
		out := RenderBlueprint("helm-deployer")
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
		expectedConfig := `
{
  "apiVersion": "helm.deployer.landscaper.gardener.cloud/v1alpha1",
  "chart": {
    "ref": "eu.gcr.io/gardener-project/landscaper/charts/helm-deployer-controller:v0.5.3"
  },
  "helmDeployment": false,
  "kind": "ProviderConfiguration",
  "name": "landscaper-helm-deployer",
  "namespace": "helm-deployer",
  "updateStrategy": "update",
  "values": {
    "deployer": {
      "namespace": "",
      "oci": {
        "allowPlainHttp": false,
        "secrets": {}
      }
    },
    "image": {
      "pullPolicy": "IfNotPresent",
      "repository": "eu.gcr.io/gardener-project/landscaper/helm-deployer-controller",
      "tag": "v0.5.3"
    },
    "replicaCount": 1
  }
}
`
		Expect(di.Spec.Configuration.Raw).To(MatchJSON(expectedConfig))
	})

	It("ManifestDeployer", func() {
		out := RenderBlueprint("manifest-deployer")
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
		expectedConfig := `
{
  "apiVersion": "helm.deployer.landscaper.gardener.cloud/v1alpha1",
  "chart": {
    "ref": "eu.gcr.io/gardener-project/landscaper/charts/manifest-deployer-controller:v0.5.3"
  },
  "helmDeployment": false,
  "kind": "ProviderConfiguration",
  "name": "landscaper-manifest-deployer",
  "namespace": "manifest-deployer",
  "updateStrategy": "update",
  "values": {
    "deployer": {
      "namespace": "",
      "oci": {
        "allowPlainHttp": false,
        "secrets": {}
      }
    },
    "image": {
      "pullPolicy": "IfNotPresent",
      "repository": "eu.gcr.io/gardener-project/landscaper/manifest-deployer-controller",
      "tag": "v0.5.3"
    },
    "replicaCount": 1
  }
}
`
		Expect(di.Spec.Configuration.Raw).To(MatchJSON(expectedConfig))
	})

	It("MockDeployer", func() {
		out := RenderBlueprint("mock-deployer")
		Expect(out.DeployItems).To(HaveLen(1))
		Expect(out.Installations).To(HaveLen(0))

		di := out.DeployItems[0]
		Expect(di.Spec.Type).To(Equal(helm.Type))
		expectedConfig := `
{
  "apiVersion": "helm.deployer.landscaper.gardener.cloud/v1alpha1",
  "chart": {
    "ref": "eu.gcr.io/gardener-project/landscaper/charts/mock-deployer-controller:v0.5.3"
  },
  "helmDeployment": false,
  "kind": "ProviderConfiguration",
  "name": "landscaper-mock-deployer",
  "namespace": "mock-deployer",
  "updateStrategy": "update",
  "values": {
    "deployer": {
      "namespace": "",
      "oci": {
        "allowPlainHttp": false,
        "secrets": {}
      }
    },
    "image": {
      "pullPolicy": "IfNotPresent",
      "repository": "eu.gcr.io/gardener-project/landscaper/mock-deployer-controller",
      "tag": "v0.5.3"
    },
    "replicaCount": 1
  }
}
`
		Expect(di.Spec.Configuration.Raw).To(MatchJSON(expectedConfig))
	})

})
