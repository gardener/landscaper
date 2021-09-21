// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcemanager_test

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Exporter", func() {

	var (
		ctx     context.Context
		cancel  context.CancelFunc
		state   *envtest.State
		timeout = 20 * time.Second
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var err error
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		cancel()
		Expect(state.CleanupState(context.TODO()))
	})

	It("should export from a existing configmap", func() {
		ctx := context.Background()
		cm := &corev1.ConfigMap{}
		cm.Name = "my-data"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"somekey": "abc",
		}
		Expect(state.Create(ctx, cm)).To(Succeed())

		exports := &managedresource.Exports{
			Exports: []managedresource.Export{
				{
					Key:      "exportkey",
					JSONPath: "data.somekey",
					FromResource: &lsv1alpha1.TypedObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
					},
				},
			},
		}
		res, err := resourcemanager.NewExporter(logr.Discard(), resourcemanager.ExporterOptions{
			KubeClient: testenv.Client,
			Objects: managedresource.ManagedResourceStatusList{
				{
					Resource: corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						Name:       cm.Name,
						Namespace:  cm.Namespace,
					},
				},
			},
		}).Export(ctx, exports)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(map[string]interface{}{
			"exportkey": "abc",
		}))
	})

	It("should export from a configmap that is created after some time", func() {
		ctx := context.Background()
		cm := &corev1.ConfigMap{}
		cm.Name = "my-data"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"somekey": "abc",
		}

		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
				Expect(state.Create(ctx, cm)).To(Succeed())
			}
		}()

		exports := &managedresource.Exports{
			Exports: []managedresource.Export{
				{
					Key:      "exportkey",
					JSONPath: "data.somekey",
					FromResource: &lsv1alpha1.TypedObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
					},
				},
			},
		}
		res, err := resourcemanager.NewExporter(logr.Discard(), resourcemanager.ExporterOptions{
			KubeClient: testenv.Client,
			Objects: managedresource.ManagedResourceStatusList{
				{
					Resource: corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						Name:       cm.Name,
						Namespace:  cm.Namespace,
					},
				},
			},
		}).Export(ctx, exports)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(map[string]interface{}{
			"exportkey": "abc",
		}))
	})

	It("should export from a configmap that data is updated after some time", func() {
		ctx := context.Background()
		cm := &corev1.ConfigMap{}
		cm.Name = "my-data"
		cm.Namespace = state.Namespace
		Expect(state.Create(ctx, cm)).To(Succeed())

		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				time.Sleep(5 * time.Second)
				cm.Data = map[string]string{
					"somekey": "abc",
				}
				Expect(testenv.Client.Update(ctx, cm)).To(Succeed())
			}
		}()

		exports := &managedresource.Exports{
			Exports: []managedresource.Export{
				{
					Key:      "exportkey",
					JSONPath: "data.somekey",
					FromResource: &lsv1alpha1.TypedObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
					},
				},
			},
		}
		res, err := resourcemanager.NewExporter(logr.Discard(), resourcemanager.ExporterOptions{
			KubeClient:     testenv.Client,
			DefaultTimeout: &timeout,
			Objects: managedresource.ManagedResourceStatusList{
				{
					Resource: corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						Name:       cm.Name,
						Namespace:  cm.Namespace,
					},
				},
			},
		}).Export(ctx, exports)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(map[string]interface{}{
			"exportkey": "abc",
		}))
	})

	It("should refuse to export if the resource is not managed by the exporter", func() {
		ctx := context.Background()
		cm := &corev1.ConfigMap{}
		cm.Name = "my-data"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"somekey": "abc",
		}
		Expect(state.Create(ctx, cm)).To(Succeed())

		exports := &managedresource.Exports{
			Exports: []managedresource.Export{
				{
					Key:      "exportkey",
					JSONPath: "data.somekey",
					FromResource: &lsv1alpha1.TypedObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
					},
				},
			},
		}
		_, err := resourcemanager.NewExporter(logr.Discard(), resourcemanager.ExporterOptions{
			KubeClient:     testenv.Client,
			DefaultTimeout: &timeout,
			Objects:        managedresource.ManagedResourceStatusList{},
		}).Export(ctx, exports)
		Expect(err).To(HaveOccurred())
	})

	Context("referenced resource", func() {
		It("should export from a referenced configmap", func() {
			ctx := context.Background()
			cm := &corev1.ConfigMap{}
			cm.Name = "my-data"
			cm.Namespace = state.Namespace
			cm.Data = map[string]string{
				"name":      "my-ref-data",
				"namespace": cm.Namespace,
			}
			Expect(state.Create(ctx, cm)).To(Succeed())
			refCm := &corev1.ConfigMap{}
			refCm.Name = "my-ref-data"
			refCm.Namespace = state.Namespace
			refCm.Data = map[string]string{
				"somekey": "abc",
			}
			Expect(state.Create(ctx, refCm)).To(Succeed())

			exports := &managedresource.Exports{
				Exports: []managedresource.Export{
					{
						Key:      "exportkey",
						JSONPath: "data",
						FromResource: &lsv1alpha1.TypedObjectReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							ObjectReference: lsv1alpha1.ObjectReference{
								Name:      cm.Name,
								Namespace: cm.Namespace,
							},
						},
						FromObjectReference: &managedresource.FromObjectReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							JSONPath:   "data.somekey",
						},
					},
				},
			}
			res, err := resourcemanager.NewExporter(logr.Discard(), resourcemanager.ExporterOptions{
				KubeClient:     testenv.Client,
				DefaultTimeout: &timeout,
				Objects: managedresource.ManagedResourceStatusList{
					{
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       cm.Name,
							Namespace:  cm.Namespace,
						},
					},
				},
			}).Export(ctx, exports)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(map[string]interface{}{
				"exportkey": "abc",
			}))
		})

		It("should export from a referenced secret in an array", func() {
			ctx := context.Background()

			sa := &corev1.ServiceAccount{}
			sa.Name = "my-sa"
			sa.Namespace = state.Namespace
			sa.Secrets = []corev1.ObjectReference{
				{
					Name: "sa-token",
				},
			}
			Expect(state.Create(ctx, sa)).To(Succeed())

			secret := &corev1.Secret{}
			secret.Name = "sa-token"
			secret.Namespace = state.Namespace
			secret.Data = map[string][]byte{
				"somekey": []byte("abc"),
			}
			Expect(state.Create(ctx, secret)).To(Succeed())

			exports := &managedresource.Exports{
				Exports: []managedresource.Export{
					{
						Key:      "exportkey",
						JSONPath: "secrets[0]",
						FromResource: &lsv1alpha1.TypedObjectReference{
							APIVersion: "v1",
							Kind:       "ServiceAccount",
							ObjectReference: lsv1alpha1.ObjectReference{
								Name:      sa.Name,
								Namespace: sa.Namespace,
							},
						},
						FromObjectReference: &managedresource.FromObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							JSONPath:   "data.somekey",
						},
					},
				},
			}
			res, err := resourcemanager.NewExporter(logr.Discard(), resourcemanager.ExporterOptions{
				KubeClient:     testenv.Client,
				DefaultTimeout: &timeout,
				Objects: managedresource.ManagedResourceStatusList{
					{
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "ServiceAccount",
							Name:       sa.Name,
							Namespace:  sa.Namespace,
						},
					},
				},
			}).Export(ctx, exports)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(map[string]interface{}{
				"exportkey": base64.StdEncoding.EncodeToString([]byte("abc")),
			}))
		})
	})

})
