// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest_test

import (
	"context"
	"encoding/base64"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Export", func() {

	var (
		ctx    context.Context
		state  *envtest.State
		target *lsv1alpha1.Target
	)

	BeforeEach(func() {
		var err error

		ctx = logging.NewContext(context.Background(), logging.Discard())
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())

		target, err = utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())
	})

	AfterEach(func() {
		defer ctx.Done()
		Expect(state.CleanupState(ctx)).To(Succeed())
	})

	It("should export with \"fromResource\"", func() {
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
		exportKey := "my-cm-data"
		manifestConfig.Exports = &managedresource.Exports{
			DefaultTimeout: &lsv1alpha1.Duration{
				Duration: time.Minute * 1,
			},
			Exports: []managedresource.Export{
				{
					Key:      exportKey,
					JSONPath: ".data",
					FromResource: &lsv1alpha1.TypedObjectReference{
						APIVersion: cm.APIVersion,
						Kind:       cm.Kind,
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
					},
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
		Expect(item.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(item.Status.ExportReference).ToNot(BeNil())

		export := &corev1.Secret{}
		Expect(state.Client.Get(ctx, item.Status.ExportReference.NamespacedName(), export)).To(Succeed())
		Expect(export.Data).To(HaveKey("config"))

		var exportData map[string]interface{}
		Expect(json.Unmarshal(export.Data["config"], &exportData)).To(Succeed())
		Expect(exportData).To(HaveKey(exportKey))
		Expect(exportData[exportKey]).To(HaveKeyWithValue("key", "val"))
	})

	It("should export with \"fromResource\" and \"fromObjectRef\"", func() {
		secret := &corev1.Secret{}
		secret.Name = "my-sec"
		secret.Namespace = state.Namespace
		secret.StringData = map[string]string{
			"key": "value",
		}

		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"name":      secret.Name,
			"namespace": secret.Namespace,
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
		exportKey := "my-cm-data"
		manifestConfig.Exports = &managedresource.Exports{
			DefaultTimeout: &lsv1alpha1.Duration{
				Duration: time.Minute * 1,
			},
			Exports: []managedresource.Export{
				{
					Key:      exportKey,
					JSONPath: ".data",
					FromResource: &lsv1alpha1.TypedObjectReference{
						APIVersion: cm.APIVersion,
						Kind:       cm.Kind,
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
					},
					FromObjectReference: &managedresource.FromObjectReference{
						APIVersion: "v1",
						Kind:       "Secret",
						JSONPath:   ".data.key",
					},
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

		go func() {
			time.Sleep(1 * time.Second)
			ctx := context.Background()
			Expect(state.Client.Create(ctx, secret)).To(Succeed())
		}()

		Expect(m.Reconcile(ctx)).To(Succeed())
		Expect(item.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(item.Status.ExportReference).ToNot(BeNil())

		export := &corev1.Secret{}
		Expect(state.Client.Get(ctx, item.Status.ExportReference.NamespacedName(), export)).To(Succeed())
		Expect(export.Data).To(HaveKey("config"))

		var exportData map[string]interface{}
		Expect(json.Unmarshal(export.Data["config"], &exportData)).To(Succeed())
		Expect(exportData).To(HaveKey(exportKey))
		Expect(exportData[exportKey]).To(Equal(base64.StdEncoding.EncodeToString([]byte("value"))))
	})
})
