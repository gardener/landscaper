package helm_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"k8s.io/utils/ptr"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/helm"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
	"github.com/gardener/landscaper/test/utils/matchers"
)

var _ = Describe("Deletion Manager", func() {

	const (
		chartPath6 = "./testdata/testchart6"
		chartPath7 = "./testdata/testchart7"
		chartPath8 = "./testdata/testchart8"
		chartPath9 = "./testdata/testchart9"

		// names of test resources and crds deployed by the deployitems
		testA             = "test-a"
		testB             = "test-b"
		testC             = "test-c"
		testD             = "test-d"
		testObject        = "testobjects.landscaper.gardener.cloud"
		clusterTestObject = "clustertestobjects.landscaper.gardener.cloud"
	)

	var (
		ctx       context.Context
		cancel    context.CancelFunc
		state     *envtest.State
		ctrl      reconcile.Reconciler
		resources *resourceBuilder
		target    *lsv1alpha1.Target
		exist     func() types.GomegaMatcher
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		ctx = logging.NewContextWithDiscard(ctx)
		var err error
		state, err = envtest.InitStateWithNamespace(ctx, testenv.Client, nil, true)
		Expect(err).ToNot(HaveOccurred())
		resources = &resourceBuilder{state.Namespace}

		deployer, err := helm.NewDeployer(testenv.Client, testenv.Client, testenv.Client, testenv.Client, logging.Discard(), helmv1alpha1.Configuration{})
		Expect(err).ToNot(HaveOccurred())

		ctrl = deployerlib.NewController(
			testenv.Client, testenv.Client, testenv.Client, testenv.Client,
			utils.NewFinishedObjectCache(),
			api.LandscaperScheme,
			record.NewFakeRecorder(1024),
			api.LandscaperScheme,
			deployerlib.DeployerArgs{Type: helm.Type, Deployer: deployer},
			5, false, "deletiongroup-test"+testutils.GetNextCounter())

		timeout.ActivateIgnoreTimeoutChecker()
		exist = func() types.GomegaMatcher { return matchers.Exist(state.Client) }

		By("Create target")
		Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
		target, err = testutils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())
	})

	AfterEach(func() {
		timeout.ActivateStandardTimeoutChecker()
		cancel()
		_ = state.CleanupState(context.Background(), envtest.WithCleanupTimeout(1*time.Millisecond))
	})

	buildDeployItem := func(
		name,
		chartPath string,
		deletionGroups []managedresource.DeletionGroupDefinition,
		deletionGroupsDuringUpdate []managedresource.DeletionGroupDefinition,
	) *lsv1alpha1.DeployItem {
		By("Build deployitem")
		chartBytes, closer := testutils.ReadChartFrom(chartPath)
		defer closer()

		chartAccess := helmv1alpha1.Chart{
			Archive: &helmv1alpha1.ArchiveAccess{
				Raw: base64.StdEncoding.EncodeToString(chartBytes),
			},
		}

		values := map[string]string{"namespace2": state.Namespace2}
		valuesRaw, err := json.Marshal(values)
		Expect(err).NotTo(HaveOccurred())

		helmConfig := &helmv1alpha1.ProviderConfiguration{
			Chart:                      chartAccess,
			Name:                       name,
			Namespace:                  state.Namespace,
			CreateNamespace:            false,
			Values:                     valuesRaw,
			HelmDeployment:             ptr.To[bool](false),
			DeletionGroups:             deletionGroups,
			DeletionGroupsDuringUpdate: deletionGroupsDuringUpdate,
		}
		item, err := helm.NewDeployItemBuilder().
			Key(state.Namespace, name).
			ProviderConfig(helmConfig).
			WithTimeout(30*time.Second).
			Target(target.Namespace, target.Name).
			GenerateJobID().
			Build()
		Expect(err).ToNot(HaveOccurred())
		return item
	}

	deployDeployItem := func(item *lsv1alpha1.DeployItem) {
		Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())
		Expect(item).To(exist())
		Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
		Eventually(func(g Gomega) {
			_, err := ctrl.Reconcile(ctx, testutils.RequestFromObject(item))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
			g.Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))
		}, 10*time.Second, 1*time.Second).Should(Succeed())
	}

	updateDeployItem := func(item *lsv1alpha1.DeployItem) {
		Expect(state.Update(ctx, item)).To(Succeed())
		Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
		item.Status.SetJobID(uuid.New().String())
		Expect(state.Client.Status().Update(ctx, item)).To(Succeed())
		Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(item))
		Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
		Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))
	}

	deleteDeployItem := func(item *lsv1alpha1.DeployItem) {
		Expect(state.Client.Delete(ctx, item)).To(Succeed())
		Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
		item.Status.SetJobID(uuid.New().String())
		Expect(state.Client.Status().Update(ctx, item)).To(Succeed())
		Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(item))
		Eventually(func(g Gomega) {
			g.Expect(item).NotTo(exist())
		}, 10*time.Second, 1*time.Second).Should(Succeed())
	}

	Context("delete", func() {

		// These tests first install and then uninstall a helm chart. During uninstall, the resources of the chart
		// should be deleted in groups. For example, first the group of namespaces resources, next the group of
		// cluster-scoped resources, and finally the group of crds.

		executeDeleteTest := func(
			deletionGroups []managedresource.DeletionGroupDefinition,
			resourcesWithFinalizer []*unstructured.Unstructured,
			expectations expectations,
		) {
			item := buildDeployItem("myitem", chartPath6, deletionGroups, nil)
			deployDeployItem(item)

			// check that all resources exist
			for _, res := range expectations.deletedResources {
				Expect(res).To(exist())
			}
			for _, res := range expectations.remainingResources {
				Expect(res).To(exist())
			}

			// add finalizers
			for _, res := range resourcesWithFinalizer {
				Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(res), res)).To(Succeed())
				res.SetFinalizers([]string{"landscaper.gardener.cloud/test1", "landscaper.gardener.cloud/test2"})
				Expect(state.Client.Update(ctx, res)).To(Succeed())
			}

			deleteDeployItem(item)

			for _, res := range expectations.deletedResources {
				Expect(res).NotTo(exist())
			}
			for _, res := range expectations.remainingResources {
				Expect(res).To(exist())
			}
		}

		It("should delete default deletion groups", func() {
			executeDeleteTest(
				// No deletion groups specified. Therefore, the default groups are used: namespaced > cluster-scoped > crds
				nil,
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.serviceAccount(testB, state.Namespace),
						resources.clusterRole(testA),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
					remainingResources: nil,
				},
			)
		})

		It("should delete namespaced resources", func() {
			executeDeleteTest(
				[]managedresource.DeletionGroupDefinition{
					{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
						Type: managedresource.PredefinedResourceGroupNamespacedResources,
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.serviceAccount(testB, state.Namespace),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.clusterRole(testA),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
				},
			)
		})

		It("should delete cluster-scoped resources", func() {
			executeDeleteTest(
				[]managedresource.DeletionGroupDefinition{
					{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
						Type: managedresource.PredefinedResourceGroupClusterScopedResources,
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.clusterRole(testA),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.serviceAccount(testB, state.Namespace),
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
				},
			)
		})

		It("should delete crds", func() {
			executeDeleteTest(
				[]managedresource.DeletionGroupDefinition{
					{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
						Type: managedresource.PredefinedResourceGroupCRDs,
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.serviceAccount(testB, state.Namespace),
						resources.clusterRole(testA),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
					},
				},
			)
		})

		It("should delete no resources", func() {
			executeDeleteTest(
				[]managedresource.DeletionGroupDefinition{
					{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
						Type: managedresource.PredefinedResourceGroupEmpty,
					}},
				},
				nil,
				expectations{
					deletedResources: nil,
					remainingResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.serviceAccount(testB, state.Namespace),
						resources.clusterRole(testA),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
				},
			)
		})

		It("should delete resources of given api version and kind", func() {
			executeDeleteTest(
				[]managedresource.DeletionGroupDefinition{
					{CustomResourceGroup: &managedresource.CustomResourceGroup{
						Resources: []managedresource.ResourceType{
							resources.serviceAccountType(nil, []string{"other"}),
							resources.clusterRoleBindingType(),
							resources.clusterRoleType([]string{testA}),
						},
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
						resources.clusterRole(testA),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.serviceAccount(testB, state.Namespace),
						resources.clusterRole(testB),
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
				},
			)
		})

		It("should delete two groups", func() {
			executeDeleteTest(
				[]managedresource.DeletionGroupDefinition{
					{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
						Type: managedresource.PredefinedResourceGroupClusterScopedResources,
					}},
					{CustomResourceGroup: &managedresource.CustomResourceGroup{
						Resources: []managedresource.ResourceType{
							resources.serviceAccountType(nil, []string{state.Namespace}),
						},
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.serviceAccount(testB, state.Namespace),
						resources.clusterRole(testA),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
				},
			)
		})

		It("should force delete resources with finalizers", func() {
			executeDeleteTest(
				[]managedresource.DeletionGroupDefinition{
					{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
						Type:        managedresource.PredefinedResourceGroupNamespacedResources,
						ForceDelete: true,
					}},
					{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
						Type:        managedresource.PredefinedResourceGroupClusterScopedResources,
						ForceDelete: true,
					}},
				},
				[]*unstructured.Unstructured{
					resources.serviceAccount(testA, state.Namespace),
					resources.clusterRole(testA),
				},
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.serviceAccount(testB, state.Namespace),
						resources.clusterRole(testA),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
				},
			)
		})
	})

	Context("update", func() {

		// These tests first install and then updates a helm chart. During the update, the orphaned resources
		// should be deleted in groups.

		executeUpdateTest := func(
			deletionGroups []managedresource.DeletionGroupDefinition,
			resourcesWithFinalizer []*unstructured.Unstructured,
			expectations expectations,
		) {
			item := buildDeployItem("myitem", chartPath6, nil, deletionGroups)
			deployDeployItem(item)

			// check that all resources exist
			for _, res := range expectations.deletedResources {
				Expect(res).To(exist())
			}
			for _, res := range expectations.remainingResources {
				Expect(res).To(exist())
			}

			itemUpdated := buildDeployItem("myitem", chartPath7, nil, deletionGroups)
			itemUpdated.SetResourceVersion(item.ResourceVersion)
			updateDeployItem(itemUpdated)

			for _, res := range expectations.deletedResources {
				Expect(res).NotTo(exist())
			}
			for _, res := range expectations.remainingResources {
				Expect(res).To(exist())
			}
		}

		It("should cleanup orphaned resources of the default deletion groups", func() {
			executeUpdateTest(
				nil,
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.serviceAccount(testB, state.Namespace),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
						resources.crd(clusterTestObject),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.clusterRole(testA),
						resources.crd(testObject),
					},
				},
			)
		})

		It("should cleanup cluster-scoped orphaned resources", func() {
			executeUpdateTest(
				[]managedresource.DeletionGroupDefinition{
					{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
						Type: managedresource.PredefinedResourceGroupClusterScopedResources,
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.serviceAccount(testB, state.Namespace),
						resources.serviceAccount(testA, state.Namespace),
						resources.clusterRole(testA),
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
				},
			)
		})

		It("should cleanup resources of given api version and kind", func() {
			executeUpdateTest(
				[]managedresource.DeletionGroupDefinition{
					{CustomResourceGroup: &managedresource.CustomResourceGroup{
						Resources: []managedresource.ResourceType{
							resources.serviceAccountType(nil, nil),
						},
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						resources.serviceAccount(testB, state.Namespace),
					},
					remainingResources: []*unstructured.Unstructured{
						resources.serviceAccount(testA, state.Namespace),
						resources.clusterRole(testA),
						resources.clusterRole(testB),
						resources.clusterRoleBinding(testA),
						resources.clusterRoleBinding(testB),
						resources.crd(testObject),
						resources.crd(clusterTestObject),
					},
				},
			)
		})

	})

	Context("delete-all", func() {

		// These tests first install and then uninstall a helm chart. During uninstall, the resources of the chart
		// should be deleted in groups. For example, first the group of namespaces resources, next the group of
		// cluster-scoped resources, and finally the group of crds.

		executeDeleteAllTest := func(
			deletionGroups []managedresource.DeletionGroupDefinition,
			resourcesWithFinalizer []*unstructured.Unstructured,
			expectations expectations,
		) {
			item := buildDeployItem("myitem", chartPath8, deletionGroups, nil)
			deployDeployItem(item)

			// deploy further resources (beyond the first chart), so that we can check the delete-all feature
			item2 := buildDeployItem("myitem2", chartPath9, deletionGroups, nil)
			deployDeployItem(item2)

			// check that all resources exist
			for _, res := range expectations.deletedResources {
				Expect(res).To(exist())
			}
			for _, res := range expectations.remainingResources {
				Expect(res).To(exist())
			}

			deleteDeployItem(item)

			for _, res := range expectations.deletedResources {
				Expect(res).NotTo(exist())
			}
			for _, res := range expectations.remainingResources {
				Expect(res).To(exist())
			}
		}

		It("should delete all resources of given types", func() {
			executeDeleteAllTest(
				[]managedresource.DeletionGroupDefinition{
					{CustomResourceGroup: &managedresource.CustomResourceGroup{
						Resources: []managedresource.ResourceType{
							resources.testObjectType(nil, nil),
							resources.clusterTestObjectType(nil),
						},
						DeleteAllResources: true,
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						// chart 8
						resources.testObject(testA, state.Namespace),
						resources.testObject(testB, state.Namespace),
						resources.testObject(testC, state.Namespace2),
						resources.testObject(testD, state.Namespace2),
						resources.clusterTestObject(testA),
						resources.clusterTestObject(testB),
						// chart 9
						resources.testObject(testA, state.Namespace2),
						resources.testObject(testB, state.Namespace2),
						resources.testObject(testC, state.Namespace),
						resources.testObject(testD, state.Namespace),
						resources.clusterTestObject(testC),
						resources.clusterTestObject(testD),
					},
					remainingResources: []*unstructured.Unstructured{},
				},
			)
		})

		It("should delete all resources of given types and namespaces", func() {
			executeDeleteAllTest(
				[]managedresource.DeletionGroupDefinition{
					{CustomResourceGroup: &managedresource.CustomResourceGroup{
						Resources: []managedresource.ResourceType{
							resources.testObjectType(nil, []string{state.Namespace}),
							resources.clusterTestObjectType(nil),
						},
						DeleteAllResources: true,
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						// chart 8
						resources.testObject(testA, state.Namespace),
						resources.testObject(testB, state.Namespace),
						resources.clusterTestObject(testA),
						resources.clusterTestObject(testB),
						// chart 9
						resources.testObject(testC, state.Namespace),
						resources.testObject(testD, state.Namespace),
						resources.clusterTestObject(testC),
						resources.clusterTestObject(testD),
					},
					remainingResources: []*unstructured.Unstructured{
						// chart 8
						resources.testObject(testC, state.Namespace2),
						resources.testObject(testD, state.Namespace2),
						// chart 9
						resources.testObject(testA, state.Namespace2),
						resources.testObject(testB, state.Namespace2),
					},
				},
			)
		})

		It("should delete all resources of given types and names", func() {
			executeDeleteAllTest(
				[]managedresource.DeletionGroupDefinition{
					{CustomResourceGroup: &managedresource.CustomResourceGroup{
						Resources: []managedresource.ResourceType{
							resources.testObjectType([]string{testB, testC}, nil),
							resources.clusterTestObjectType([]string{testB, testD}),
						},
						DeleteAllResources: true,
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						// chart 8
						resources.testObject(testB, state.Namespace),
						resources.testObject(testC, state.Namespace2),
						resources.clusterTestObject(testB),
						// chart 9
						resources.testObject(testB, state.Namespace2),
						resources.testObject(testC, state.Namespace),
						resources.clusterTestObject(testD),
					},
					remainingResources: []*unstructured.Unstructured{
						// chart 8
						resources.testObject(testA, state.Namespace),
						resources.testObject(testD, state.Namespace2),
						resources.clusterTestObject(testA),
						// chart 9
						resources.testObject(testA, state.Namespace2),
						resources.testObject(testD, state.Namespace),
						resources.clusterTestObject(testC),
					},
				},
			)
		})

		It("should delete all resources of given types, names, and namespaces", func() {
			executeDeleteAllTest(
				[]managedresource.DeletionGroupDefinition{
					{CustomResourceGroup: &managedresource.CustomResourceGroup{
						Resources: []managedresource.ResourceType{
							resources.testObjectType([]string{testA, testC}, []string{state.Namespace2}),
						},
						DeleteAllResources: true,
					}},
				},
				nil,
				expectations{
					deletedResources: []*unstructured.Unstructured{
						// chart 8
						resources.testObject(testC, state.Namespace2),
						// chart 9
						resources.testObject(testA, state.Namespace2),
					},
					remainingResources: []*unstructured.Unstructured{
						// chart 8
						resources.testObject(testA, state.Namespace),
						resources.testObject(testB, state.Namespace),
						resources.testObject(testD, state.Namespace2),
						resources.clusterTestObject(testA),
						resources.clusterTestObject(testB),
						// chart 9
						resources.testObject(testB, state.Namespace2),
						resources.testObject(testC, state.Namespace),
						resources.testObject(testD, state.Namespace),
						resources.clusterTestObject(testC),
						resources.clusterTestObject(testD),
					},
				},
			)
		})
	})

})

type resourceBuilder struct {
	namespace string
}

func (r *resourceBuilder) resource(apiVersion, kind, name, namespace string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)
	u.SetName(name)
	u.SetNamespace(namespace)
	return u
}

func (r *resourceBuilder) serviceAccount(name, namespace string) *unstructured.Unstructured {
	return r.resource("v1", "ServiceAccount", name, namespace)
}

func (r *resourceBuilder) serviceAccountType(names, namespaces []string) managedresource.ResourceType {
	return managedresource.ResourceType{
		APIVersion: "v1",
		Kind:       "ServiceAccount",
		Names:      names,
		Namespaces: namespaces,
	}
}

func (r *resourceBuilder) clusterRole(name string) *unstructured.Unstructured {
	return r.resource("rbac.authorization.k8s.io/v1", "ClusterRole", name, "")
}

func (r *resourceBuilder) clusterRoleType(names []string) managedresource.ResourceType {
	return managedresource.ResourceType{
		APIVersion: "rbac.authorization.k8s.io/v1",
		Kind:       "ClusterRole",
		Names:      names,
	}
}

func (r *resourceBuilder) clusterRoleBinding(name string) *unstructured.Unstructured {
	return r.resource("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", name, "")
}

func (r *resourceBuilder) clusterRoleBindingType() managedresource.ResourceType {
	return managedresource.ResourceType{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRoleBinding"}
}

func (r *resourceBuilder) crd(name string) *unstructured.Unstructured {
	return r.resource("apiextensions.k8s.io/v1", "CustomResourceDefinition", name, "")
}

func (r *resourceBuilder) testObject(name, namespace string) *unstructured.Unstructured {
	return r.resource("landscaper.gardener.cloud/v1alpha1", "TestObject", name, namespace)
}

func (r *resourceBuilder) testObjectType(names, namespaces []string) managedresource.ResourceType {
	return managedresource.ResourceType{
		APIVersion: "landscaper.gardener.cloud/v1alpha1",
		Kind:       "TestObject",
		Names:      names,
		Namespaces: namespaces,
	}
}

func (r *resourceBuilder) clusterTestObject(name string) *unstructured.Unstructured {
	return r.resource("landscaper.gardener.cloud/v1alpha1", "ClusterTestObject", name, "")
}

func (r *resourceBuilder) clusterTestObjectType(names []string) managedresource.ResourceType {
	return managedresource.ResourceType{
		APIVersion: "landscaper.gardener.cloud/v1alpha1",
		Kind:       "ClusterTestObject",
		Names:      names,
	}
}

type expectations struct {
	deletedResources   []*unstructured.Unstructured
	remainingResources []*unstructured.Unstructured
}
