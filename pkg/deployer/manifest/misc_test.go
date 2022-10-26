// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest_test

import (
	"context"
	"github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
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
			lsv1alpha1.DefaultKubeconfigKey: kubeconfigBytes,
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

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())

		cmRes := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), cmRes)).To(Succeed())
		Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
	})

	It("should fail if the secret ref has another namespace than the target", func() {
		secretNamespace := state.Namespace + "alt"
		secretNamespaceObj := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretNamespace,
			},
		}
		Expect(state.Create(ctx, secretNamespaceObj)).To(Succeed())

		kubeconfigBytes, err := kutil.GenerateKubeconfigBytes(testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		kSecret := &corev1.Secret{}
		kSecret.Name = "my-target"
		kSecret.Namespace = secretNamespace
		kSecret.Data = map[string][]byte{
			lsv1alpha1.DefaultKubeconfigKey: kubeconfigBytes,
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

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).NotTo(Succeed())

		Expect(state.Client.Delete(ctx, secretNamespaceObj)).To(Succeed())
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

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).ToNot(HaveKeyWithValue("to-be-deleted", "True"))

		Expect(m.Delete(ctx)).ToNot(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).To(HaveKeyWithValue("to-be-deleted", "True"))
		Expect(cm.ObjectMeta.DeletionTimestamp).ToNot(BeNil())

		cm.ObjectMeta.Finalizers = nil
		Expect(state.Client.Update(ctx, cm)).To(Succeed())

		Eventually(func() error {
			return m.Delete(ctx)
		}, 1*time.Second).WithTimeout(1 * time.Minute).Should(Succeed())
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

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).ToNot(HaveKeyWithValue("to-be-deleted", "True"))
		Expect(cm.Annotations).To(HaveKeyWithValue("always", "True"))

		Expect(m.Delete(ctx)).ToNot(Succeed())
		Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
		Expect(cm.Annotations).To(HaveKeyWithValue("to-be-deleted", "True"))
		Expect(cm.Annotations).To(HaveKeyWithValue("always", "True"))
		Expect(cm.ObjectMeta.DeletionTimestamp).ToNot(BeNil())

		cm.ObjectMeta.Finalizers = nil
		Expect(state.Client.Update(ctx, cm)).To(Succeed())

		Eventually(func() error {
			return m.Delete(ctx)
		}, 1*time.Second).WithTimeout(1 * time.Minute).Should(Succeed())
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

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
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

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
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

		manifestConfig := &manifestv1alpha2.ProviderConfiguration{
			ReadinessChecks: readinesschecks.ReadinessCheckConfiguration{
				Timeout: &lsv1alpha1.Duration{
					Duration: 1 * time.Second,
				},
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Timeout: &lsv1alpha1.Duration{
							Duration: 1 * time.Minute,
						},
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
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())

		m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			time.Sleep(2 * time.Second)
			Expect(state.Client.Get(ctx, k8sclient.ObjectKeyFromObject(cm), cm)).To(Succeed())
			cm.Data["ready"] = "true"
			Expect(state.Client.Update(ctx, cm)).To(Succeed())
		}()
		Expect(m.Reconcile(ctx)).To(Succeed())
	})
})
