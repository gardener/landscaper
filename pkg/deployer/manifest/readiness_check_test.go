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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("ReadinessCheck", func() {
	var (
		ctx    context.Context
		state  *envtest.State
		target *lsv1alpha1.ResolvedTarget
	)

	BeforeEach(func() {
		var err error

		ctx = logging.NewContext(context.Background(), logging.Discard())
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())

		rawTarget, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, rawTarget)).To(Succeed())
		target = lsv1alpha1.NewResolvedTarget(rawTarget)
	})

	AfterEach(func() {
		defer ctx.Done()
		Expect(state.CleanupState(ctx)).To(Succeed())
	})

	Context("labelSelector", func() {
		It("should mark the deploy item as ready for a managed configmap", func() {
			managedConfigMap := &corev1.ConfigMap{}
			managedConfigMap.Name = "my-managed-cm"
			managedConfigMap.Namespace = state.Namespace
			managedConfigMap.Data = map[string]string{
				"status": "ready",
				"key":    "val",
			}
			managedConfigMap.Labels = map[string]string{
				"app": "test",
			}

			rawCM, err := kutil.ConvertToRawExtension(managedConfigMap, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
			manifestConfig.Manifests = []managedresource.Manifest{
				{
					Policy:   managedresource.ManagePolicy,
					Manifest: rawCM,
				},
			}

			requirementValue := map[string]string{
				"value": "ready",
			}
			requirementValueMarshaled, err := json.Marshal(requirementValue)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig.ReadinessChecks = readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "secret-selector",
						LabelSelector: &readinesschecks.LabelSelectorSpec{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Labels:     managedConfigMap.Labels,
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.status",
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
			}
			deployItem, err := manifest.NewDeployItemBuilder().
				Key(state.Namespace, "my-cm").
				ProviderConfig(manifestConfig).
				Target(target.Namespace, target.Name).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, deployItem)).To(Succeed())
			Expect(state.SetInitTime(ctx, deployItem)).To(Succeed())

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(managedConfigMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})
	})

	Context("Operator Equals", func() {
		It("should mark the deploy item as ready for a managed configmap", func() {
			managedConfigMap := &corev1.ConfigMap{}
			managedConfigMap.Name = "my-managed-cm"
			managedConfigMap.Namespace = state.Namespace
			managedConfigMap.Data = map[string]string{
				"status": "ready",
				"key":    "val",
			}

			rawCM, err := kutil.ConvertToRawExtension(managedConfigMap, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
			manifestConfig.Manifests = []managedresource.Manifest{
				{
					Policy:   managedresource.ManagePolicy,
					Manifest: rawCM,
				},
			}

			requirementValue := map[string]string{
				"value": "ready",
			}
			requirementValueMarshaled, err := json.Marshal(requirementValue)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig.ReadinessChecks = readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "secret-selector",
						Resource: []lsv1alpha1.TypedObjectReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								ObjectReference: lsv1alpha1.ObjectReference{
									Name:      "my-managed-cm",
									Namespace: state.Namespace,
								},
							},
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.status",
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
			}
			deployItem, err := manifest.NewDeployItemBuilder().
				Key(state.Namespace, "my-cm").
				ProviderConfig(manifestConfig).
				Target(target.Namespace, target.Name).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, deployItem)).To(Succeed())
			Expect(state.SetInitTime(ctx, deployItem)).To(Succeed())

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(managedConfigMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})

		It("should mark the deploy item as not ready for a managed configmap with an invalid value", func() {
			managedConfigMap := &corev1.ConfigMap{}
			managedConfigMap.Name = "my-managed-cm"
			managedConfigMap.Namespace = state.Namespace
			managedConfigMap.Data = map[string]string{
				"status": "notReady",
				"key":    "val",
			}

			rawCM, err := kutil.ConvertToRawExtension(managedConfigMap, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
			manifestConfig.Manifests = []managedresource.Manifest{
				{
					Policy:   managedresource.ManagePolicy,
					Manifest: rawCM,
				},
			}

			requirementValue := map[string]string{
				"value": "ready",
			}
			requirementValueMarshaled, err := json.Marshal(requirementValue)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig.ReadinessChecks = readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "secret-selector",
						Resource: []lsv1alpha1.TypedObjectReference{
							{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								ObjectReference: lsv1alpha1.ObjectReference{
									Name:      "my-managed-cm",
									Namespace: state.Namespace,
								},
							},
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.status",
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
			}
			deployItem, err := manifest.NewDeployItemBuilder().
				Key(state.Namespace, "my-cm").
				ProviderConfig(manifestConfig).
				Target(target.Namespace, target.Name).
				WithTimeout(1 * time.Second).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, deployItem)).To(Succeed())
			Expect(state.SetInitTime(ctx, deployItem)).To(Succeed())

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).ToNot(Succeed())
		})
	})

	Context("Operator NotEquals", func() {
		It("should mark the deploy item as ready for a managed configmap", func() {
			managedConfigMap := &corev1.ConfigMap{}
			managedConfigMap.Name = "my-managed-cm"
			managedConfigMap.Namespace = state.Namespace
			managedConfigMap.Data = map[string]string{
				"status": "ready",
				"key":    "val",
			}
			managedConfigMap.Labels = map[string]string{
				"app": "test",
			}

			rawCM, err := kutil.ConvertToRawExtension(managedConfigMap, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
			manifestConfig.Manifests = []managedresource.Manifest{
				{
					Policy:   managedresource.ManagePolicy,
					Manifest: rawCM,
				},
			}

			requirementValue := map[string]string{
				"value": "bad",
			}
			requirementValueMarshaled, err := json.Marshal(requirementValue)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig.ReadinessChecks = readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "secret-selector",
						LabelSelector: &readinesschecks.LabelSelectorSpec{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Labels:     managedConfigMap.Labels,
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.status",
								Operator: selection.NotEquals,
								Value: []runtime.RawExtension{
									{
										Raw: requirementValueMarshaled,
									},
								},
							},
						},
					},
				},
			}
			deployItem, err := manifest.NewDeployItemBuilder().
				Key(state.Namespace, "my-cm").
				ProviderConfig(manifestConfig).
				Target(target.Namespace, target.Name).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, deployItem)).To(Succeed())
			Expect(state.SetInitTime(ctx, deployItem)).To(Succeed())

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(managedConfigMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})
	})

	Context("Operator In", func() {
		It("should mark the deploy item as ready for a managed configmap", func() {
			managedConfigMap := &corev1.ConfigMap{}
			managedConfigMap.Name = "my-managed-cm"
			managedConfigMap.Namespace = state.Namespace
			managedConfigMap.Data = map[string]string{
				"status": "ready",
				"key":    "val",
			}
			managedConfigMap.Labels = map[string]string{
				"app": "test",
			}

			rawCM, err := kutil.ConvertToRawExtension(managedConfigMap, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
			manifestConfig.Manifests = []managedresource.Manifest{
				{
					Policy:   managedresource.ManagePolicy,
					Manifest: rawCM,
				},
			}

			requirementValue1 := map[string]string{
				"value": "ok",
			}
			requirementValueMarshaled1, err := json.Marshal(requirementValue1)
			Expect(err).ToNot(HaveOccurred())

			requirementValue2 := map[string]string{
				"value": "ready",
			}
			requirementValueMarshaled2, err := json.Marshal(requirementValue2)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig.ReadinessChecks = readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "secret-selector",
						LabelSelector: &readinesschecks.LabelSelectorSpec{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Labels:     managedConfigMap.Labels,
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.status",
								Operator: selection.In,
								Value: []runtime.RawExtension{
									{
										Raw: requirementValueMarshaled1,
									},
									{
										Raw: requirementValueMarshaled2,
									},
								},
							},
						},
					},
				},
			}
			deployItem, err := manifest.NewDeployItemBuilder().
				Key(state.Namespace, "my-cm").
				ProviderConfig(manifestConfig).
				Target(target.Namespace, target.Name).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, deployItem)).To(Succeed())
			Expect(state.SetInitTime(ctx, deployItem)).To(Succeed())

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(managedConfigMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})
	})

	Context("Operator NotIn", func() {
		It("should mark the deploy item as ready for a managed configmap", func() {
			managedConfigMap := &corev1.ConfigMap{}
			managedConfigMap.Name = "my-managed-cm"
			managedConfigMap.Namespace = state.Namespace
			managedConfigMap.Data = map[string]string{
				"status": "ready",
				"key":    "val",
			}
			managedConfigMap.Labels = map[string]string{
				"app": "test",
			}

			rawCM, err := kutil.ConvertToRawExtension(managedConfigMap, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
			manifestConfig.Manifests = []managedresource.Manifest{
				{
					Policy:   managedresource.ManagePolicy,
					Manifest: rawCM,
				},
			}

			requirementValue1 := map[string]string{
				"value": "bad",
			}
			requirementValueMarshaled1, err := json.Marshal(requirementValue1)
			Expect(err).ToNot(HaveOccurred())

			requirementValue2 := map[string]string{
				"value": "pending",
			}
			requirementValueMarshaled2, err := json.Marshal(requirementValue2)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig.ReadinessChecks = readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "secret-selector",
						LabelSelector: &readinesschecks.LabelSelectorSpec{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Labels:     managedConfigMap.Labels,
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.status",
								Operator: selection.NotIn,
								Value: []runtime.RawExtension{
									{
										Raw: requirementValueMarshaled1,
									},
									{
										Raw: requirementValueMarshaled2,
									},
								},
							},
						},
					},
				},
			}
			deployItem, err := manifest.NewDeployItemBuilder().
				Key(state.Namespace, "my-cm").
				ProviderConfig(manifestConfig).
				Target(target.Namespace, target.Name).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, deployItem)).To(Succeed())
			Expect(state.SetInitTime(ctx, deployItem)).To(Succeed())

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(managedConfigMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})
	})

	Context("Operator Exists", func() {
		It("should mark the deploy item as ready for a managed configmap", func() {
			managedConfigMap := &corev1.ConfigMap{}
			managedConfigMap.Name = "my-managed-cm"
			managedConfigMap.Namespace = state.Namespace
			managedConfigMap.Data = map[string]string{
				"status": "ready",
				"key":    "val",
			}
			managedConfigMap.Labels = map[string]string{
				"app": "test",
			}

			rawCM, err := kutil.ConvertToRawExtension(managedConfigMap, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
			manifestConfig.Manifests = []managedresource.Manifest{
				{
					Policy:   managedresource.ManagePolicy,
					Manifest: rawCM,
				},
			}

			requirementValue := map[string]string{
				"value": "any",
			}
			requirementValueMarshaled, err := json.Marshal(requirementValue)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig.ReadinessChecks = readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "secret-selector",
						LabelSelector: &readinesschecks.LabelSelectorSpec{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Labels:     managedConfigMap.Labels,
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.status",
								Operator: selection.Exists,
								Value: []runtime.RawExtension{
									{
										Raw: requirementValueMarshaled,
									},
								},
							},
						},
					},
				},
			}
			deployItem, err := manifest.NewDeployItemBuilder().
				Key(state.Namespace, "my-cm").
				ProviderConfig(manifestConfig).
				Target(target.Namespace, target.Name).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, deployItem)).To(Succeed())
			Expect(state.SetInitTime(ctx, deployItem)).To(Succeed())

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(managedConfigMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})
	})

	Context("Operator DoesNotExist", func() {
		It("should mark the deploy item as ready for a managed configmap", func() {
			managedConfigMap := &corev1.ConfigMap{}
			managedConfigMap.Name = "my-managed-cm"
			managedConfigMap.Namespace = state.Namespace
			managedConfigMap.Data = map[string]string{
				"status": "ready",
				"key":    "val",
			}
			managedConfigMap.Labels = map[string]string{
				"app": "test",
			}

			rawCM, err := kutil.ConvertToRawExtension(managedConfigMap, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig := &manifestv1alpha2.ProviderConfiguration{}
			manifestConfig.Manifests = []managedresource.Manifest{
				{
					Policy:   managedresource.ManagePolicy,
					Manifest: rawCM,
				},
			}

			requirementValue := map[string]string{
				"value": "any",
			}
			requirementValueMarshaled, err := json.Marshal(requirementValue)
			Expect(err).ToNot(HaveOccurred())

			manifestConfig.ReadinessChecks = readinesschecks.ReadinessCheckConfiguration{
				CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
					{
						Name: "secret-selector",
						LabelSelector: &readinesschecks.LabelSelectorSpec{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Labels:     managedConfigMap.Labels,
						},
						Requirements: []readinesschecks.RequirementSpec{
							{
								JsonPath: ".data.invalid",
								Operator: selection.DoesNotExist,
								Value: []runtime.RawExtension{
									{
										Raw: requirementValueMarshaled,
									},
								},
							},
						},
					},
				},
			}
			deployItem, err := manifest.NewDeployItemBuilder().
				Key(state.Namespace, "my-cm").
				ProviderConfig(manifestConfig).
				Target(target.Namespace, target.Name).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, deployItem)).To(Succeed())
			Expect(state.SetInitTime(ctx, deployItem)).To(Succeed())

			m, err := manifest.New(testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, deployItem, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Reconcile(ctx)).To(Succeed())

			cmRes := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(managedConfigMap), cmRes)).To(Succeed())
			Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
		})
	})
})
