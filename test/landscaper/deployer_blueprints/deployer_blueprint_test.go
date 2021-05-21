// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployer_blueprints_test

import (
	"path/filepath"
	"testing"

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

var _ = Describe("Blueprint", func() {

	It("ContainerDeployer", func() {
		out, err := lsutils.RenderBlueprint(lsutils.BlueprintRenderArgs{
			Fs:      osfs.New(),
			RootDir: filepath.Join(projectRoot, ".landscaper/container-deployer"),
		})
		testutils.ExpectNoError(err)
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
		out, err := lsutils.RenderBlueprint(lsutils.BlueprintRenderArgs{
			Fs:      osfs.New(),
			RootDir: filepath.Join(projectRoot, ".landscaper/helm-deployer"),
		})
		testutils.ExpectNoError(err)
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
		out, err := lsutils.RenderBlueprint(lsutils.BlueprintRenderArgs{
			Fs:      osfs.New(),
			RootDir: filepath.Join(projectRoot, ".landscaper/manifest-deployer"),
		})
		testutils.ExpectNoError(err)
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
		out, err := lsutils.RenderBlueprint(lsutils.BlueprintRenderArgs{
			Fs:      osfs.New(),
			RootDir: filepath.Join(projectRoot, ".landscaper/mock-deployer"),
		})
		testutils.ExpectNoError(err)
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
