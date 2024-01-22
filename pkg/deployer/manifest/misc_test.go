// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	genericresolver "github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver/generic"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("", func() {

	var (
		ctx   context.Context
		state *envtest.State
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		var err error
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		defer ctx.Done()
		Expect(state.CleanupState(ctx)).To(Succeed())
	})

	It("should create a configured configmap with a kubeconfig target referenced by secret", func() {
		kubeconfigBytes, err := kutil.GenerateKubeconfigBytes(testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		kSecret := &corev1.Secret{}
		kSecret.Name = "my-target"
		kSecret.Namespace = state.Namespace
		kSecret.Data = map[string][]byte{
			targettypes.DefaultKubeconfigKey: kubeconfigBytes,
		}
		Expect(state.Create(ctx, kSecret)).To(Succeed())

		target, err := utils.CreateKubernetesTargetFromSecret(state.Namespace, "my-target", kSecret)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"key": "val",
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		sr := genericresolver.New(state.Client)
		rt, err := sr.Resolve(ctx, target)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
		manifestConfig.Manifests = []managedresource.Manifest{
			{
				Policy:   managedresource.ManagePolicy,
				Manifest: rawCM,
			},
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())
		Expect(state.SetInitTime(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, rt)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())

		cmRes := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), cmRes)).To(Succeed())
		Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
	})

	It("should add before delete annotations when manifest is being deleted", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cm",
				Namespace: state.Namespace,
				Finalizers: []string{
					"kubernetes.io/test",
				},
			},
			Data: map[string]string{
				"key": "val",
			},
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		sr := genericresolver.New(state.Client)
		rt, err := sr.Resolve(ctx, target)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
		manifestConfig.Manifests = []managedresource.Manifest{
			{
				Policy:   managedresource.ManagePolicy,
				Manifest: rawCM,
				AnnotateBeforeDelete: map[string]string{
					"to-be-deleted": "True",
				},
			},
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			WithTimeout(12 * time.Second).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())
		Expect(state.SetInitTime(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, rt)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).ToNot(HaveKeyWithValue("to-be-deleted", "True"))

		// We delete the deployitem. This should add a before-delete annotation and a deletion timestamp at the deployed
		// configmap. A finalizer prevents that the configmap vanishes. This allows us to check the configmap in
		// parallel. As soon as the before-delete annotation and a deletion timestamp are there, we remove
		// the finalizer, so that the deletion can finish. (The Delete method does not return until the deployitem is
		// gone or the timeout is reached.)
		go func() {
			defer GinkgoRecover()
			Eventually(func(g Gomega) {
				obj := &corev1.ConfigMap{}
				g.Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), obj)).To(Succeed())
				g.Expect(obj.Annotations).To(HaveKeyWithValue("to-be-deleted", "True"))
				g.Expect(obj.DeletionTimestamp.IsZero()).To(BeFalse())

				obj.ObjectMeta.Finalizers = nil
				g.Expect(state.Client.Update(ctx, obj)).To(Succeed())
			}, 10*time.Second, time.Second).Should(Succeed())
		}()

		Expect(m.Delete(ctx)).To(Succeed())

		err = state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "configmap should be deleted")
	})

	It("should add before delete annotations when manifest is being deleted (with additional annotations)", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cm",
				Namespace: state.Namespace,
				Finalizers: []string{
					"kubernetes.io/test",
				},
				Annotations: map[string]string{
					"always": "True",
				},
			},
			Data: map[string]string{
				"key": "val",
			},
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		sr := genericresolver.New(state.Client)
		rt, err := sr.Resolve(ctx, target)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
		manifestConfig.Manifests = []managedresource.Manifest{
			{
				Policy:   managedresource.ManagePolicy,
				Manifest: rawCM,
				AnnotateBeforeDelete: map[string]string{
					"to-be-deleted": "True",
				},
			},
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, rt)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.SetInitTime(ctx, item)).To(Succeed())

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).ToNot(HaveKeyWithValue("to-be-deleted", "True"))
		Expect(cm.Annotations).To(HaveKeyWithValue("always", "True"))

		go func() {
			defer GinkgoRecover()
			Eventually(func(g Gomega) {
				obj := &corev1.ConfigMap{}
				g.Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), obj)).To(Succeed())
				g.Expect(obj.Annotations).To(HaveKeyWithValue("to-be-deleted", "True"))
				g.Expect(obj.Annotations).To(HaveKeyWithValue("always", "True"))
				g.Expect(obj.DeletionTimestamp.IsZero()).To(BeFalse())

				obj.ObjectMeta.Finalizers = nil
				g.Expect(state.Client.Update(ctx, obj)).To(Succeed())
			}, 10*time.Second, time.Second).Should(Succeed())
		}()

		Expect(m.Delete(ctx)).To(Succeed())

		err = state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "configmap should be deleted")
	})

	It("should add before create annotations when manifest is being created", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cm",
				Namespace: state.Namespace,
			},
			Data: map[string]string{
				"key": "val",
			},
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		sr := genericresolver.New(state.Client)
		rt, err := sr.Resolve(ctx, target)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
		manifestConfig.Manifests = []managedresource.Manifest{
			{
				Policy:   managedresource.ManagePolicy,
				Manifest: rawCM,
				AnnotateBeforeCreate: map[string]string{
					"init": "True",
				},
			},
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())
		Expect(state.SetInitTime(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, rt)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).To(HaveKeyWithValue("init", "True"))

		cm.Annotations = nil
		Expect(state.Client.Update(ctx, cm))

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).ToNot(HaveKeyWithValue("init", "True"))
	})

	It("should add before create annotations when manifest is being created (with additional annotations)", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cm",
				Namespace: state.Namespace,
				Annotations: map[string]string{
					"always": "True",
				},
			},
			Data: map[string]string{
				"key": "val",
			},
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		sr := genericresolver.New(state.Client)
		rt, err := sr.Resolve(ctx, target)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
		manifestConfig.Manifests = []managedresource.Manifest{
			{
				Policy:   managedresource.ManagePolicy,
				Manifest: rawCM,
				AnnotateBeforeCreate: map[string]string{
					"init": "True",
				},
			},
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())
		Expect(state.SetInitTime(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, rt)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).To(HaveKeyWithValue("init", "True"))
		Expect(cm.Annotations).To(HaveKeyWithValue("always", "True"))

		cm.Annotations = nil
		Expect(state.Client.Update(ctx, cm))

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).ToNot(HaveKeyWithValue("init", "True"))
		Expect(cm.Annotations).To(HaveKeyWithValue("always", "True"))
	})

	It("should respect the custom readiness check timeout when set", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cm",
				Namespace: state.Namespace,
			},
			Data: map[string]string{
				"key": "val",
			},
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		requirementValue := map[string]string{
			"value": "true",
		}
		requirementValueMarshaled, err := json.Marshal(requirementValue)
		Expect(err).ToNot(HaveOccurred())

		sr := genericresolver.New(state.Client)
		rt, err := sr.Resolve(ctx, target)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{
			ReadinessChecks: readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "my-check",
						Resource: []lsv1alpha1.TypedObjectReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								ObjectReference: lsv1alpha1.ObjectReference{
									Name:      "my-cm",
									Namespace: state.Namespace,
								},
							},
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.ready",
								Operator: selection.Equals,
								Value: []runtime.RawExtension{
									{
										Raw: requirementValueMarshaled,
									},
								},
							},
						},
					},
				},
			},
		}
		manifestConfig.Manifests = []managedresource.Manifest{
			{
				Policy:   managedresource.ManagePolicy,
				Manifest: rawCM,
			},
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			WithTimeout(1 * time.Minute).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())
		Expect(state.SetInitTime(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, rt)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			time.Sleep(1 * time.Second)
			Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
			cm.Data["ready"] = "true"
			Expect(state.Client.Update(ctx, cm)).To(Succeed())
		}()
		Expect(m.Reconcile(ctx)).To(Succeed())
	})

	It("should deploy a configmap list", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cmList := &corev1.ConfigMapList{
			Items: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-cm-0",
						Namespace: state.Namespace,
						Finalizers: []string{
							"kubernetes.io/test",
						},
					},
					Data: map[string]string{
						"key0": "val0",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-cm-1",
						Namespace: state.Namespace,
						Finalizers: []string{
							"kubernetes.io/test",
						},
					},
					Data: map[string]string{
						"key1": "val1",
					},
				},
			},
		}
		rawCMList, err := kutil.ConvertToRawExtension(cmList, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		sr := genericresolver.New(state.Client)
		rt, err := sr.Resolve(ctx, target)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
		manifestConfig.Manifests = []managedresource.Manifest{
			{
				Policy:   managedresource.ManagePolicy,
				Manifest: rawCMList,
				AnnotateBeforeDelete: map[string]string{
					"to-be-deleted": "True",
				},
			},
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())
		Expect(state.SetInitTime(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, rt)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())

		cm := &corev1.ConfigMap{}
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(&cmList.Items[0]), cm)).To(Succeed())
		Expect(cm.Data).To(HaveKeyWithValue("key0", "val0"))

		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(&cmList.Items[1]), cm)).To(Succeed())
		Expect(cm.Data).To(HaveKeyWithValue("key1", "val1"))

		go func() {
			defer GinkgoRecover()
			Eventually(func(g Gomega) {
				for _, cmListItem := range cmList.Items {
					obj := &corev1.ConfigMap{}
					Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(&cmListItem), obj)).To(Succeed())
					g.Expect(obj.Annotations).To(HaveKeyWithValue("to-be-deleted", "True"))
					g.Expect(obj.DeletionTimestamp.IsZero()).To(BeFalse())

					obj.ObjectMeta.Finalizers = nil
					g.Expect(state.Client.Update(ctx, obj)).To(Succeed())
				}
			}, time.Minute, time.Second).Should(Succeed())
		}()

		Expect(m.Delete(ctx)).To(Succeed())
	})

	It("should deploy an empty configmap list", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cmList := &corev1.ConfigMapList{
			Items: []corev1.ConfigMap{},
		}
		rawCMList, err := kutil.ConvertToRawExtension(cmList, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		sr := genericresolver.New(state.Client)
		rt, err := sr.Resolve(ctx, target)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
		manifestConfig.Manifests = []managedresource.Manifest{
			{
				Policy:   managedresource.ManagePolicy,
				Manifest: rawCMList,
				AnnotateBeforeDelete: map[string]string{
					"to-be-deleted": "True",
				},
			},
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())
		Expect(state.SetInitTime(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, rt)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())

		Eventually(func() error {
			return m.Delete(ctx)
		}, 1*time.Second).WithTimeout(1 * time.Minute).Should(Succeed())
	})
})
