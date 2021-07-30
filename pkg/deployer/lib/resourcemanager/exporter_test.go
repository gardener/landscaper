// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcemanager_test

import (
	"context"
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

	var state *envtest.State

	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(state.CleanupState(context.TODO(), testenv.Client, nil))
	})

	It("should export from a existing configmap", func() {
		ctx := context.Background()
		cm := &corev1.ConfigMap{}
		cm.Name = "my-data"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"somekey": "abc",
		}
		Expect(state.Create(ctx, testenv.Client, cm)).To(Succeed())

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
					Resource: lsv1alpha1.TypedObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
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
			time.Sleep(10 * time.Second)
			Expect(state.Create(ctx, testenv.Client, cm)).To(Succeed())
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
					Resource: lsv1alpha1.TypedObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
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
		Expect(state.Create(ctx, testenv.Client, cm)).To(Succeed())

		go func() {
			time.Sleep(5 * time.Second)
			cm.Data = map[string]string{
				"somekey": "abc",
			}
			Expect(testenv.Client.Update(ctx, cm)).To(Succeed())
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
					Resource: lsv1alpha1.TypedObjectReference{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      cm.Name,
							Namespace: cm.Namespace,
						},
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
		Expect(state.Create(ctx, testenv.Client, cm)).To(Succeed())

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
			KubeClient: testenv.Client,
			Objects:    managedresource.ManagedResourceStatusList{},
		}).Export(ctx, exports)
		Expect(err).To(HaveOccurred())
	})

})
