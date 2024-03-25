// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"
	"fmt"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/deployer/helm"
	"github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/pkg/deployer/mock"
	commonutils "github.com/gardener/landscaper/pkg/utils"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func DeployerBlueprintTests(f *framework.Framework) {

	// Each of the following tests creates a Target ("my-cluster-target") and an Installation to deploy a deployer.
	// This is done for all deployers: mock, manifest, container, helm. When the deployer has been deployed,
	// we create another Target ("my-cluster-target-2") and a test DeployItem to check that the deployer works.
	//
	// Note that during the test there are two competing deployers: the one deployed by the Installation in this test,
	// and the corresponding default deployer in namespace ls-system which we use throughout the whole test suite.
	// We use target selectors to separate their responsibilities. Otherwise, the test DeployItem would be processed by
	// both of them. The default deployer is responsible for Targets without a "landscaper.gardener.cloud/environment"
	// annotation (see script hack/int-test-helper/install-landscaper, which installs the default landscaper for the
	// integration test suite and defines the target selector in the helm values). On the other hand, our new deployer
	// is only responsible for Targets with the annotation "landscaper.gardener.cloud/environment: my-deployer-env".
	Describe("Deployer Blueprint", func() {

		var (
			ctx   context.Context
			state = f.Register()
		)

		BeforeEach(func() {
			log, err := logging.GetLogger()
			if err != nil {
				f.Log().Logfln("Error fetching logger: %w", err)
				return
			}
			ctx = logging.NewContext(context.Background(), log)
		})

		AfterEach(func() {
			defer ctx.Done()
		})

		It("MockDeployer should deploy a deployer with its blueprint", func() {
			td := &testDefinition{
				Name:                    "MockDeployer",
				ComponentDescriptorName: "github.com/gardener/landscaper/mock-deployer",
				BlueprintResourceName:   "mock-deployer-blueprint",
				DeployItemBuilder: mock.NewDeployItemBuilder().
					ProviderConfig(&mockv1alpha1.ProviderConfiguration{
						Phase: &lsv1alpha1.DeployItemPhases.Succeeded,
					}),
			}
			executeTest(ctx, f, state, td)
		})

		It("ManifestDeployer should deploy a deployer with its blueprint", func() {
			td := &testDefinition{
				Name:                    "ManifestDeployer",
				ComponentDescriptorName: "github.com/gardener/landscaper/manifest-deployer",
				BlueprintResourceName:   "manifest-deployer-blueprint",
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
							Manifests: []managedresource.Manifest{
								{
									Policy:   managedresource.ManagePolicy,
									Manifest: secret,
								},
							},
						}).Build()
				},
			}
			executeTest(ctx, f, state, td)
		})

		It("ContainerDeployer should deploy a deployer with its blueprint", func() {
			td := &testDefinition{
				Name:                    "ContainerDeployer",
				ComponentDescriptorName: "github.com/gardener/landscaper/container-deployer",
				BlueprintResourceName:   "container-deployer-blueprint",
				DeployItemBuilder: container.NewDeployItemBuilder().
					ProviderConfig(&containerv1alpha1.ProviderConfiguration{
						Image:   "alpine:latest",
						Command: []string{"sh", "-c"},
						Args:    []string{"echo test"},
					}),
			}
			executeTest(ctx, f, state, td)
		})

		It("HelmDeployer should deploy a deployer with its blueprint", func() {
			td := &testDefinition{
				Name:                    "HelmDeployer",
				ComponentDescriptorName: "github.com/gardener/landscaper/helm-deployer",
				BlueprintResourceName:   "helm-deployer-blueprint",
				DeployItem: func(state *envtest.State, target *lsv1alpha1.Target) (*lsv1alpha1.DeployItem, error) {
					ref := "eu.gcr.io/gardener-project/landscaper/integration-tests/charts/hello-world:1.0.0"
					return helm.NewDeployItemBuilder().
						Key(state.Namespace, "my-di").
						Target(target.Namespace, target.Name).
						ProviderConfig(&helmv1alpha1.ProviderConfiguration{
							Name:      "my-chart",
							Namespace: state.Namespace,
							Chart: helmv1alpha1.Chart{
								Ref: ref,
							},
						}).Build()
				},
			}
			executeTest(ctx, f, state, td)
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

func executeTest(ctx context.Context, f *framework.Framework, state *framework.State, td *testDefinition) {
	By("Create Target for the installation")
	// This target has no "landscaper.gardener.cloud/environment" annotation, so that a default deployer is responsible.
	targetName := "my-cluster-target"
	target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, targetName, f.RestConfig)
	utils.ExpectNoError(err)
	utils.ExpectNoError(state.Create(ctx, target))

	By("Create Configuration for the installation")
	cm := &corev1.ConfigMap{}
	cm.Name = "deployer-config"
	cm.Namespace = state.Namespace
	cm.Data = map[string]string{
		"releaseName":      "my-deployer",
		"releaseNamespace": state.Namespace,
		// add target selector to not interfere with default deployer
		"values": `{"deployer":{"targetSelector":[{"annotations":[{"key":"landscaper.gardener.cloud/environment","operator":"=","values":["my-deployer-env"]}]}]}}`,
	}
	utils.ExpectNoError(state.Create(ctx, cm))

	// build installation
	repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository(framework.OpenSourceRepositoryContext, ""))
	utils.ExpectNoError(err)
	inst := &lsv1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployer",
			Namespace: state.Namespace,
			Annotations: map[string]string{
				lsv1alpha1.OperationAnnotation: string(lsv1alpha1.ReconcileOperation),
			},
		},
		Spec: lsv1alpha1.InstallationSpec{
			ComponentDescriptor: &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					RepositoryContext: &repoCtx,
					ComponentName:     td.ComponentDescriptorName,
					Version:           f.LsVersion,
				},
			},
			Blueprint: lsv1alpha1.BlueprintDefinition{
				Reference: &lsv1alpha1.RemoteBlueprintReference{
					ResourceName: td.BlueprintResourceName,
				},
			},
			Imports: lsv1alpha1.InstallationImports{
				Targets: []lsv1alpha1.TargetImport{
					{
						Name:   "cluster",
						Target: "#" + target.Name,
					},
				},
				Data: []lsv1alpha1.DataImport{
					{
						Name: "releaseName",
						ConfigMapRef: &lsv1alpha1.LocalConfigMapReference{
							Name: cm.Name,
							Key:  "releaseName",
						},
					},
					{
						Name: "releaseNamespace",
						ConfigMapRef: &lsv1alpha1.LocalConfigMapReference{
							Name: cm.Name,
							Key:  "releaseNamespace",
						},
					},
					{
						Name: "values",
						ConfigMapRef: &lsv1alpha1.LocalConfigMapReference{
							Name: cm.Name,
							Key:  "values",
						},
					},
				},
			},
		},
	}

	utils.ExpectNoError(state.Create(ctx, inst))
	utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 10*time.Minute))

	By("Testing the deployer with a simple deployitem")
	// This target has a "landscaper.gardener.cloud/environment" annotation, so that the deployer is responsible which we have just installed.
	targetName2 := "my-cluster-target-2"
	target2, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, targetName2, f.RestConfig)
	utils.ExpectNoError(err)
	annotations := target2.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}
	annotations["landscaper.gardener.cloud/environment"] = "my-deployer-env"
	target2.SetAnnotations(annotations)
	utils.ExpectNoError(state.Create(ctx, target2))

	var di *lsv1alpha1.DeployItem
	if td.DeployItemBuilder != nil {
		di, err = td.DeployItemBuilder.
			Key(state.Namespace, "deployer-di-test").
			Target(target2.Namespace, target2.Name).
			Build()
		utils.ExpectNoError(err)
	} else if td.DeployItem != nil {
		di, err = td.DeployItem(state.State, target2)
		utils.ExpectNoError(err)
	}
	Expect(di).ToNot(BeNil())

	itemStringInit := "Found DI init: " + fmt.Sprintf("%+v\n", *di)

	utils.ExpectNoError(state.Create(ctx, di))

	itemStringCreate := "Found DI create: " + fmt.Sprintf("%+v\n", *di)

	// Set a new jobID to trigger a reconcile of the deploy item
	utils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))
	itemStringBefore := "Found DI before: " + fmt.Sprintf("%+v\n", *di)

	err = utils.UpdateJobIdForDeployItemC(ctx, f.Client, di, metav1.Now())

	if err != nil {
		f.TestLog().Logfln(itemStringInit)
		f.TestLog().Logfln(itemStringCreate)
		f.TestLog().Logfln(itemStringBefore)
		utils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))
		f.TestLog().Logfln("Found DI after: " + fmt.Sprintf("%+v\n", *di))
	}

	utils.ExpectNoError(err)

	By("Waiting for deploy item " + di.GetName() + " to succeed")
	utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhases.Succeeded, 3*time.Minute))

	By("Delete deploy item for new reconcile")
	utils.ExpectNoError(utils.DeleteDeployItemForNewReconcile(ctx, f.Client, di, 3*time.Minute))

	By("Delete installation")
	utils.ExpectNoError(utils.DeleteObject(ctx, f.Client, inst, 3*time.Minute))
}
