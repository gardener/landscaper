// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"
	"fmt"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/helm"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/container"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/pkg/deployer/mock"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	commonutils "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func DeployerBlueprintTests(f *framework.Framework) {
	ginkgo.Describe("Deployer Blueprint", func() {

		phase := lsv1alpha1.ExecutionPhaseSucceeded
		TestDeployerBlueprint(f, testDefinition{
			Name:                    "MockDeployer",
			ComponentDescriptorName: "github.com/gardener/landscaper/mock-deployer",
			BlueprintResourceName:   "mock-deployer-blueprint",
			DeployItemBuilder: mock.NewDeployItemBuilder().
				ProviderConfig(&mockv1alpha1.ProviderConfiguration{
					Phase: &phase,
				}),
		})

		TestDeployerBlueprint(f, testDefinition{
			Name:                    "ManifestDeployer",
			ComponentDescriptorName: "github.com/gardener/landscaper/mock-deployer",
			BlueprintResourceName:   "mock-deployer-blueprint",
			DeployItem: func(state *envtest.State, target *lsv1alpha1.Target) (*lsv1alpha1.DeployItem, error) {
				secret, _ := kutil.ConvertToRawExtension(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "my-secret", Namespace: state.Namespace},
					Data: map[string][]byte{
						"key": []byte("val"),
					},
				}, scheme.Scheme)
				return manifest.NewDeployItemBuilder().
					Key(state.Namespace, "my-di").
					Target(target.Namespace, target.Name).
					ProviderConfig(&manifestv1alpha2.ProviderConfiguration{
						Manifests: []manifestv1alpha2.Manifest{
							{
								Policy:   manifestv1alpha2.ManagePolicy,
								Manifest: secret,
							},
						},
					}).Build()
			},
		})

		TestDeployerBlueprint(f, testDefinition{
			Name:                    "ContainerDeployer",
			ComponentDescriptorName: "github.com/gardener/landscaper/container-deployer",
			BlueprintResourceName:   "container-deployer-blueprint",
			DeployItemBuilder: container.NewDeployItemBuilder().
				ProviderConfig(&containerv1alpha1.ProviderConfiguration{
					Image:   "alpine:latest",
					Command: []string{"sh", "-c"},
					Args:    []string{"echo test"},
				}),
		})

		TestDeployerBlueprint(f, testDefinition{
			Name:                    "HelmDeployer",
			ComponentDescriptorName: "github.com/gardener/landscaper/helm-deployer",
			BlueprintResourceName:   "helm-deployer-blueprint",
			DeployItem: func(state *envtest.State, target *lsv1alpha1.Target) (*lsv1alpha1.DeployItem, error) {
				return helm.NewDeployItemBuilder().
					Key(state.Namespace, "my-di").
					Target(target.Namespace, target.Name).
					ProviderConfig(&helmv1alpha1.ProviderConfiguration{
						Name:      "my-chart",
						Namespace: state.Namespace,
						Chart: helmv1alpha1.Chart{
							Ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v0.1.0",
						},
					}).Build()
			},
		})
	})
}

type testDefinition struct {
	Name                    string
	ComponentDescriptorName string
	BlueprintResourceName   string
	DeployItem              func(state *envtest.State, target *lsv1alpha1.Target) (*lsv1alpha1.DeployItem, error)
	DeployItemBuilder       *commonutils.DeployItemBuilder
}

func TestDeployerBlueprint(f *framework.Framework, td testDefinition) {
	var (
		state = f.Register()
		ctx   context.Context

		name                    = td.Name
		componentDescriptorName = td.ComponentDescriptorName
		blueprintResourceName   = td.BlueprintResourceName
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
	})

	ginkgo.AfterEach(func() {
		defer ctx.Done()
	})

	ginkgo.It(fmt.Sprintf("[%s] should deploy a deployer with its blueprint", name), func() {
		ginkgo.By("Create Target for the installation")
		target := &lsv1alpha1.Target{}
		target.Name = "my-cluster-target"
		target.Namespace = state.Namespace
		target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
		utils.ExpectNoError(err)
		utils.ExpectNoError(state.Create(ctx, f.Client, target))

		ginkgo.By("Create Configuration for the installation")
		cm := &corev1.ConfigMap{}
		cm.Name = "deployer-config"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"releaseName":      "my-deployer",
			"releaseNamespace": state.Namespace,
			// todo: add own target selector to not interfere with other tests
			"values": "{}",
		}
		utils.ExpectNoError(state.Create(ctx, f.Client, cm))
		cmRef := lsv1alpha1.ObjectReference{
			Name:      cm.Name,
			Namespace: cm.Namespace,
		}

		// build installation
		repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository(framework.OpenSourceRepositoryContext, ""))
		utils.ExpectNoError(err)
		inst := &lsv1alpha1.Installation{}
		inst.Name = "deployer"
		inst.Namespace = state.Namespace
		inst.Annotations = map[string]string{
			lsv1alpha1.NotUseDefaultDeployerAnnotation: "true",
		}
		inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
			Reference: &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: &repoCtx,
				ComponentName:     componentDescriptorName,
				Version:           f.LsVersion,
			},
		}
		inst.Spec.Blueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
			ResourceName: blueprintResourceName,
		}
		inst.Spec.Imports.Targets = []lsv1alpha1.TargetImport{
			{
				Name:   "cluster",
				Target: "#" + target.Name,
			},
		}
		inst.Spec.Imports.Data = []lsv1alpha1.DataImport{
			{
				Name: "releaseName",
				ConfigMapRef: &lsv1alpha1.ConfigMapReference{
					ObjectReference: cmRef,
					Key:             "releaseName",
				},
			},
			{
				Name: "releaseNamespace",
				ConfigMapRef: &lsv1alpha1.ConfigMapReference{
					ObjectReference: cmRef,
					Key:             "releaseNamespace",
				},
			},
			{
				Name: "values",
				ConfigMapRef: &lsv1alpha1.ConfigMapReference{
					ObjectReference: cmRef,
					Key:             "values",
				},
			},
		}

		utils.ExpectNoError(state.Create(ctx, f.Client, inst))
		utils.ExpectNoError(lsutils.WaitForInstallationToBeHealthy(ctx, f.Client, inst, 2*time.Minute))

		ginkgo.By("Testing the deployer with a simple deployitem")

		var di *lsv1alpha1.DeployItem
		if td.DeployItemBuilder != nil {
			di, err = td.DeployItemBuilder.
				Key(state.Namespace, "deployer-di-test").
				Target(target.Namespace, target.Name).
				Build()
			utils.ExpectNoError(err)
		} else if td.DeployItem != nil {
			di, err = td.DeployItem(&state.State, target)
			utils.ExpectNoError(err)
		}
		g.Expect(di).ToNot(g.BeNil())
		utils.ExpectNoError(state.Create(ctx, f.Client, di))

		utils.ExpectNoError(lsutils.WaitForDeployItemToSucceed(ctx, f.Client, di, 2*time.Minute))

		utils.ExpectNoError(utils.DeleteObject(ctx, f.Client, di, 2*time.Minute))
		utils.ExpectNoError(utils.DeleteObject(ctx, f.Client, inst, 2*time.Minute))
	})
}
