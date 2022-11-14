// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetresolver"
	"github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func createDeployItem(ctx context.Context, state *envtest.State, name string, target *lsv1alpha1.Target, configMap *corev1.ConfigMap, policy managedresource.ManifestPolicy) *lsv1alpha1.DeployItem {
	rawCM, err := kutil.ConvertToRawExtension(configMap, scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())

	manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
	manifestConfig.Manifests = []managedresource.Manifest{
		{
			Policy:   policy,
			Manifest: rawCM,
		},
	}
	deployItem, err := manifest.NewDeployItemBuilder().
		Key(state.Namespace, name).
		ProviderConfig(manifestConfig).
		Target(target.Namespace, target.Name).
		Build()
	Expect(err).ToNot(HaveOccurred())
	Expect(state.Create(ctx, deployItem)).To(Succeed())

	return deployItem
}

func updateDeployItem(ctx context.Context, state *envtest.State, deployItem *lsv1alpha1.DeployItem, configMap *corev1.ConfigMap, policy managedresource.ManifestPolicy) {
	Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(deployItem), deployItem)).To(Succeed())

	rawCM, err := kutil.ConvertToRawExtension(configMap, scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())

	manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
	manifestConfig.Manifests = []managedresource.Manifest{
		{
			Policy:   policy,
			Manifest: rawCM,
		},
	}

	rawConfig, err := kutil.ConvertToRawExtension(manifestConfig, manifest.Scheme)
	Expect(err).ToNot(HaveOccurred())
	deployItem.Spec.Configuration = rawConfig
	Expect(state.Client.Update(ctx, deployItem)).ToNot(HaveOccurred())
}

var _ = Describe("Policy", func() {

	var (
		ctx       context.Context
		state     *envtest.State
		configMap *corev1.ConfigMap
		target    *lsv1alpha1.ResolvedTarget
	)

	BeforeEach(func() {
		var err error

		ctx = logging.NewContext(context.Background(), logging.Discard())
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())

		rawTarget, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, rawTarget)).To(Succeed())
		target = targetresolver.NewResolvedTarget(rawTarget)

		configMap = &corev1.ConfigMap{}
		configMap.Name = "my-cm"
		configMap.Namespace = state.Namespace
		configMap.Data = map[string]string{
			"key": "val",
		}
	})

	AfterEach(func() {
		defer ctx.Done()
		Expect(state.CleanupState(ctx)).To(Succeed())
	})

	Context("Manage", func() {
		It("should create a configured configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.ManagePolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})

		It("should update the created configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.ManagePolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))

			configMap.Data = map[string]string{
				"key": "updated",
			}
			updateDeployItem(ctx, state, deployItem, configMap, managedresource.ManagePolicy)
			m, err = manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "updated"))
		})

		It("should delete a created configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.ManagePolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())

			Expect(m.Delete(ctx)).To(HaveOccurred())
			Expect(m.Delete(ctx)).To(Succeed())

			cmRes = &corev1.ConfigMap{}
			Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(configMap), cmRes))).To(BeTrue())
		})
	})

	Context("Fallback", func() {
		It("should create a configured configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.FallbackPolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})

		It("should update the created configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.FallbackPolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))

			configMap.Data = map[string]string{
				"key": "updated",
			}
			updateDeployItem(ctx, state, deployItem, configMap, managedresource.FallbackPolicy)
			m, err = manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "updated"))
		})

		It("should not update the created configmap when the deployer label is not matching", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.FallbackPolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))

			cmRes.Labels[manifestv1alpha2.ManagedDeployItemLabel] = "invalid"
			Expect(testenv.Client.Update(ctx, cmRes)).To(Succeed())

			configMap.Data = map[string]string{
				"key": "updated",
			}
			updateDeployItem(ctx, state, deployItem, configMap, managedresource.FallbackPolicy)
			m, err = manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})

		It("should delete a created configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.FallbackPolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())

			Expect(m.Delete(ctx)).To(HaveOccurred())
			Expect(m.Delete(ctx)).To(Succeed())

			cmRes = &corev1.ConfigMap{}
			Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(configMap), cmRes))).To(BeTrue())
		})

		It("should not delete a created configmap when the deployer label is not matching", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.FallbackPolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())

			cmRes.Labels[manifestv1alpha2.ManagedDeployItemLabel] = "invalid"
			Expect(testenv.Client.Update(ctx, cmRes)).To(Succeed())

			Expect(m.Delete(ctx)).To(Succeed())

			cmRes = &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
		})
	})

	Context("Keep", func() {
		It("should create a configured configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.KeepPolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})

		It("should update the created configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.KeepPolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))

			configMap.Data = map[string]string{
				"key": "updated",
			}
			updateDeployItem(ctx, state, deployItem, configMap, managedresource.KeepPolicy)
			m, err = manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "updated"))
		})

		It("should not delete a created configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.KeepPolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())

			Expect(m.Delete(ctx)).To(Succeed())

			cmRes = &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
		})
	})

	Context("Ignore", func() {
		It("should not create a configured configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.IgnorePolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(configMap), cmRes))).To(BeTrue())
		})
	})

	Context("Immutable", func() {
		It("should create a configured configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.ImmutablePolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})

		It("should not update the created configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.ImmutablePolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))

			configMap.Data = map[string]string{
				"key": "updated",
			}
			updateDeployItem(ctx, state, deployItem, configMap, managedresource.ImmutablePolicy)
			m, err = manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})

		It("should delete a created configmap", func() {
			deployItem := createDeployItem(ctx, state, "my-deployitem", target.Target, configMap, managedresource.ImmutablePolicy)

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(configMap), cmRes)).To(Succeed())

			Expect(m.Delete(ctx)).To(HaveOccurred())
			Expect(m.Delete(ctx)).To(Succeed())

			cmRes = &corev1.ConfigMap{}
			Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(configMap), cmRes))).To(BeTrue())
		})
	})
})
