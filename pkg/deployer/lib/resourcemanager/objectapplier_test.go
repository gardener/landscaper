// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcemanager_test

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/utils/envtest"

	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("ObjectApplier", func() {

	var (
		state *envtest.State
	)

	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(state.CleanupState(context.TODO(), testenv.Client, nil))
	})

	It("should apply and update a configmap", func() {
		ctx := context.TODO()
		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"key": "val",
		}
		cmRaw, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		opts := resourcemanager.ManifestApplierOptions{
			Decoder:          api.NewDecoder(scheme.Scheme),
			KubeClient:       testenv.Client,
			Clientset:        clientset,
			DefaultNamespace: state.Namespace,
			DeleteTimeout:    10 * time.Second,
			UpdateStrategy:   manifestv1alpha2.UpdateStrategyUpdate,
			Manifests: []managedresource.Manifest{
				{
					Manifest: cmRaw,
				},
			},
			ManagedResources: managedresource.ManagedResourceStatusList{},
		}

		managedResources, err := resourcemanager.ApplyManifests(ctx, logr.Discard(), opts)
		Expect(err).ToNot(HaveOccurred())

		res := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), res)).To(Succeed())
		Expect(res.Data).To(HaveKeyWithValue("key", "val"))
		Expect(managedResources).To(HaveLen(1))
		Expect(managedResources[0].Resource).To(Equal(corev1.ObjectReference{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       "my-cm",
			Namespace:  state.Namespace,
			UID:        res.UID,
		}))

		cm.Data["key"] = "modified"
		cmRaw, err = kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())
		opts.Manifests = []managedresource.Manifest{
			{
				Manifest: cmRaw,
			},
		}
		opts.ManagedResources = managedResources
		_, err = resourcemanager.ApplyManifests(ctx, logr.Discard(), opts)
		Expect(err).ToNot(HaveOccurred())

		res = &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), res)).To(Succeed())
		Expect(res.Data).To(HaveKeyWithValue("key", "modified"))
	})

	It("should delete a orphaned resource", func() {
		ctx := context.TODO()
		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"key": "val",
		}
		cmRaw, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		opts := resourcemanager.ManifestApplierOptions{
			Decoder:          api.NewDecoder(scheme.Scheme),
			KubeClient:       testenv.Client,
			Clientset:        clientset,
			DefaultNamespace: state.Namespace,
			DeleteTimeout:    10 * time.Second,
			UpdateStrategy:   manifestv1alpha2.UpdateStrategyUpdate,
			Manifests: []managedresource.Manifest{
				{
					Manifest: cmRaw,
				},
			},
			ManagedResources: managedresource.ManagedResourceStatusList{},
		}
		managedResources, err := resourcemanager.ApplyManifests(ctx, logr.Discard(), opts)
		Expect(err).ToNot(HaveOccurred())

		res := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), res)).To(Succeed())
		Expect(res.Data).To(HaveKeyWithValue("key", "val"))
		Expect(managedResources).To(HaveLen(1))
		Expect(managedResources[0].Resource).To(Equal(corev1.ObjectReference{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       "my-cm",
			Namespace:  state.Namespace,
			UID:        res.UID,
		}))

		opts.Manifests = []managedresource.Manifest{}
		opts.ManagedResources = managedResources
		managedResources, err = resourcemanager.ApplyManifests(ctx, logr.Discard(), opts)
		Expect(err).ToNot(HaveOccurred())

		res = &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), res)).To(HaveOccurred())
		Expect(managedResources).To(HaveLen(0))
	})

	It("should keep a sorted list of managed resources", func() {
		ctx := context.TODO()

		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"key": "val",
		}
		cmRaw, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())
		secret := &corev1.Secret{}
		secret.Name = "my-secret"
		secret.Namespace = state.Namespace
		secret.Data = map[string][]byte{
			"key": []byte("val"),
		}
		secretRaw, err := kutil.ConvertToRawExtension(secret, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		opts := resourcemanager.ManifestApplierOptions{
			Decoder:          api.NewDecoder(scheme.Scheme),
			KubeClient:       testenv.Client,
			Clientset:        clientset,
			DefaultNamespace: state.Namespace,
			DeleteTimeout:    10 * time.Second,
			UpdateStrategy:   manifestv1alpha2.UpdateStrategyUpdate,
			Manifests: []managedresource.Manifest{
				{
					Manifest: cmRaw,
				},
				{
					Manifest: secretRaw,
				},
			},
			ManagedResources: managedresource.ManagedResourceStatusList{},
		}
		managedResources, err := resourcemanager.ApplyManifests(ctx, logr.Discard(), opts)
		Expect(err).ToNot(HaveOccurred())

		res := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), res)).To(Succeed())
		Expect(res.Data).To(HaveKeyWithValue("key", "val"))

		Expect(managedResources).To(HaveLen(2))
		Expect(managedResources[0].Resource.Name).To(Equal("my-cm"))
		Expect(managedResources[1].Resource.Name).To(Equal("my-secret"))

		for i := 0; i > 10; i++ {
			cm.Data["key"] = fmt.Sprintf("modified-%d", i)
			cmRaw, err = kutil.ConvertToRawExtension(cm, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			opts.Manifests[0].Manifest = cmRaw
			opts.ManagedResources = managedResources
			managedResources, err = resourcemanager.ApplyManifests(ctx, logr.Discard(), opts)
			Expect(err).ToNot(HaveOccurred())

			Expect(managedResources).To(HaveLen(2))
			Expect(managedResources[0].Resource.Name).To(Equal("my-cm"))
			Expect(managedResources[1].Resource.Name).To(Equal("my-secret"))
		}

	})

	It("should create a namespace before other resources", func() {
		ctx := context.TODO()

		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = "test"
		cm.Data = map[string]string{
			"key": "val",
		}
		cmRaw, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())
		ns := &corev1.Namespace{}
		ns.Name = "test"

		nsRaw, err := kutil.ConvertToRawExtension(ns, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		opts := resourcemanager.ManifestApplierOptions{
			Decoder:          api.NewDecoder(scheme.Scheme),
			KubeClient:       testenv.Client,
			Clientset:        clientset,
			DefaultNamespace: state.Namespace,
			DeleteTimeout:    10 * time.Second,
			UpdateStrategy:   manifestv1alpha2.UpdateStrategyUpdate,
			Manifests: []managedresource.Manifest{
				{
					Manifest: cmRaw,
				},
				{
					Manifest: nsRaw,
				},
			},
			ManagedResources: managedresource.ManagedResourceStatusList{},
		}
		managedResources, err := resourcemanager.ApplyManifests(ctx, logr.Discard(), opts)
		Expect(err).ToNot(HaveOccurred())

		res := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), res)).To(Succeed())
		Expect(res.Data).To(HaveKeyWithValue("key", "val"))

		Expect(managedResources).To(HaveLen(2))
		Expect(managedResources[0].Resource.Name).To(Equal("test"))
		Expect(managedResources[1].Resource.Name).To(Equal("my-cm"))
	})

})
