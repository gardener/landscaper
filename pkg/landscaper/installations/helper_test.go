// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/pkg/landscaper/installations"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("helper", func() {

	Context("IsRoot", func() {

		It("should validate that a installation with a non Installation type owner is a root installation", func() {
			inst := &lsv1alpha1.Installation{}
			inst.Name = "inst"
			inst.Namespace = "default"
			inst.Labels = map[string]string{lsv1alpha1.EncompassedByLabel: "owner"}

			owner := &corev1.Secret{}
			owner.Name = "owner"
			owner.Namespace = "default"
			err := controllerutil.SetOwnerReference(owner, inst, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			isRoot := installations.IsRootInstallation(inst)
			Expect(isRoot).To(BeTrue())
		})

		It("should validate that a installation with a installation owner is not a root installation", func() {
			inst := &lsv1alpha1.Installation{}
			inst.Name = "inst"
			inst.Namespace = "default"
			inst.Labels = map[string]string{lsv1alpha1.EncompassedByLabel: "owner"}

			owner := &lsv1alpha1.Installation{}
			owner.Name = "owner"
			owner.Namespace = "default"
			err := controllerutil.SetOwnerReference(owner, inst, api.LandscaperScheme)
			Expect(err).ToNot(HaveOccurred())

			isRoot := installations.IsRootInstallation(inst)
			Expect(isRoot).To(BeFalse())
		})
	})

	Context("GetDataImport", func() {

		var (
			kubeClient client.Client
		)

		BeforeEach(func() {
			var err error
			kubeClient, _, err = envtest.NewFakeClientFromPath("")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should get an import from a dataobject", func() {
			ctx := context.Background()
			defer ctx.Done()
			data := &lsv1alpha1.DataObject{}
			data.Name = "test-do"
			data.Namespace = "default"
			data.Data = lsv1alpha1.NewAnyJSON([]byte("\"val1\""))
			Expect(kubeClient.Create(ctx, data)).To(Succeed())

			instAndImports := installations.NewInstallationAndImports(&lsv1alpha1.Installation{})
			inst := &installations.InstallationImportsAndBlueprint{
				InstallationAndImports: *instAndImports,
			}
			inst.GetInstallation().Namespace = data.Namespace
			do, owner, err := installations.GetDataImport(ctx, kubeClient, "", &inst.InstallationAndImports, lsv1alpha1.DataImport{
				Name:    "imp",
				DataRef: "#test-do",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(BeNil())
			Expect(do.Data).To(Equal("val1"))
		})

		It("should throw an error if the dataobject does not exist", func() {
			ctx := context.Background()
			defer ctx.Done()

			instAndImports := installations.NewInstallationAndImports(&lsv1alpha1.Installation{})
			inst := &installations.InstallationImportsAndBlueprint{
				InstallationAndImports: *instAndImports,
			}
			inst.GetInstallation().Namespace = "default"
			_, _, err := installations.GetDataImport(ctx, kubeClient, "", &inst.InstallationAndImports, lsv1alpha1.DataImport{
				Name:    "imp",
				DataRef: "#test-do",
			})
			Expect(err).To(HaveOccurred())
		})

		It("should get an import from a configmap", func() {
			ctx := context.Background()
			defer ctx.Done()
			cm := &corev1.ConfigMap{}
			cm.Name = "test-cm"
			cm.Namespace = "default"
			cm.Data = map[string]string{
				"key1": "\"val1\"",
			}
			Expect(kubeClient.Create(ctx, cm)).To(Succeed())

			do, owner, err := installations.GetDataImport(ctx, kubeClient, "", &installations.InstallationAndImports{}, lsv1alpha1.DataImport{
				Name: "imp",
				ConfigMapRef: &lsv1alpha1.ConfigMapReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      cm.Name,
						Namespace: cm.Namespace,
					},
					Key: "key1",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(BeNil())
			Expect(do.Data).To(Equal("val1"))
		})

		It("should get an import from a configmap's binary data", func() {
			ctx := context.Background()
			defer ctx.Done()
			cm := &corev1.ConfigMap{}
			cm.Name = "test-cm"
			cm.Namespace = "default"
			cm.BinaryData = map[string][]byte{
				"key1": []byte("\"val1\""),
			}
			Expect(kubeClient.Create(ctx, cm)).To(Succeed())

			do, owner, err := installations.GetDataImport(ctx, kubeClient, "", &installations.InstallationAndImports{}, lsv1alpha1.DataImport{
				Name: "imp",
				ConfigMapRef: &lsv1alpha1.ConfigMapReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      cm.Name,
						Namespace: cm.Namespace,
					},
					Key: "key1",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(BeNil())
			Expect(do.Data).To(Equal("val1"))
		})

		It("should get an import from a whole configmap as object", func() {
			ctx := context.Background()
			defer ctx.Done()
			cm := &corev1.ConfigMap{}
			cm.Name = "test-cm"
			cm.Namespace = "default"
			cm.Data = map[string]string{
				"key1": `"val1"`,
				"key2": `val2`,
				"key3": `["a", "b", "c"]`,
				"key4": `true`,
				"key5": `{"x": {"y": "z"}}`,
			}
			Expect(kubeClient.Create(ctx, cm)).To(Succeed())

			do, owner, err := installations.GetDataImport(ctx, kubeClient, "", &installations.InstallationAndImports{}, lsv1alpha1.DataImport{
				Name: "imp",
				ConfigMapRef: &lsv1alpha1.ConfigMapReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      cm.Name,
						Namespace: cm.Namespace,
					},
					Key: "",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(BeNil())

			Expect(do.Data).To(HaveKeyWithValue("key1", "val1"))
			Expect(do.Data).To(HaveKeyWithValue("key2", "val2"))
			Expect(do.Data).To(HaveKeyWithValue("key3", []interface{}{"a", "b", "c"}))
			Expect(do.Data).To(HaveKeyWithValue("key4", true))
			Expect(do.Data).To(HaveKeyWithValue("key5", map[string]interface{}{
				"x": map[string]interface{}{
					"y": "z",
				},
			}))
		})

		It("should get an import from the binary data of whole configmap as object", func() {
			ctx := context.Background()
			defer ctx.Done()
			cm := &corev1.ConfigMap{}
			cm.Name = "test-cm"
			cm.Namespace = "default"
			cm.BinaryData = map[string][]byte{
				"key1": []byte(`"val1"`),
				"key2": []byte(`val2`),
				"key3": []byte(`["a", "b", "c"]`),
				"key4": []byte(`true`),
				"key5": []byte(`{"x": {"y": "z"}}`),
			}
			Expect(kubeClient.Create(ctx, cm)).To(Succeed())

			do, owner, err := installations.GetDataImport(ctx, kubeClient, "", &installations.InstallationAndImports{}, lsv1alpha1.DataImport{
				Name: "imp",
				ConfigMapRef: &lsv1alpha1.ConfigMapReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      cm.Name,
						Namespace: cm.Namespace,
					},
					Key: "",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(BeNil())

			Expect(do.Data).To(HaveKeyWithValue("key1", "val1"))
			Expect(do.Data).To(HaveKeyWithValue("key2", "val2"))
			Expect(do.Data).To(HaveKeyWithValue("key3", []interface{}{"a", "b", "c"}))
			Expect(do.Data).To(HaveKeyWithValue("key4", true))
			Expect(do.Data).To(HaveKeyWithValue("key5", map[string]interface{}{
				"x": map[string]interface{}{
					"y": "z",
				},
			}))
		})

		It("should throw an error if the imported key of a configmap does not exist", func() {
			ctx := context.Background()
			defer ctx.Done()
			cm := &corev1.ConfigMap{}
			cm.Name = "test-cm"
			cm.Namespace = "default"
			cm.Data = map[string]string{
				"key1": "\"val1\"",
			}
			Expect(kubeClient.Create(ctx, cm)).To(Succeed())

			_, _, err := installations.GetDataImport(ctx, kubeClient, "", &installations.InstallationAndImports{}, lsv1alpha1.DataImport{
				Name: "imp",
				ConfigMapRef: &lsv1alpha1.ConfigMapReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      cm.Name,
						Namespace: cm.Namespace,
					},
					Key: "key2",
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("should get an import from a secret", func() {
			ctx := context.Background()
			defer ctx.Done()
			secret := &corev1.Secret{}
			secret.Name = "test-secret"
			secret.Namespace = "default"
			secret.Data = map[string][]byte{
				"key1": []byte("\"val1\""),
			}
			Expect(kubeClient.Create(ctx, secret)).To(Succeed())

			do, owner, err := installations.GetDataImport(ctx, kubeClient, "", &installations.InstallationAndImports{}, lsv1alpha1.DataImport{
				Name: "imp",
				SecretRef: &lsv1alpha1.SecretReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      secret.Name,
						Namespace: secret.Namespace,
					},
					Key: "key1",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(BeNil())
			Expect(do.Data).To(Equal("val1"))
		})

		It("should get an import from a whole secret", func() {
			ctx := context.Background()
			defer ctx.Done()
			secret := &corev1.Secret{}
			secret.Name = "test-secret"
			secret.Namespace = "default"
			secret.Data = map[string][]byte{
				"key1": []byte(`"val1"`),
				"key2": []byte(`val2`),
				"key3": []byte(`["a", "b", "c"]`),
				"key4": []byte(`true`),
				"key5": []byte(`{"x": {"y": "z"}}`),
			}
			Expect(kubeClient.Create(ctx, secret)).To(Succeed())

			do, owner, err := installations.GetDataImport(ctx, kubeClient, "", &installations.InstallationAndImports{}, lsv1alpha1.DataImport{
				Name: "imp",
				SecretRef: &lsv1alpha1.SecretReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      secret.Name,
						Namespace: secret.Namespace,
					},
					Key: "",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(BeNil())

			Expect(do.Data).To(HaveKeyWithValue("key1", "val1"))
			Expect(do.Data).To(HaveKeyWithValue("key2", "val2"))
			Expect(do.Data).To(HaveKeyWithValue("key3", []interface{}{"a", "b", "c"}))
			Expect(do.Data).To(HaveKeyWithValue("key4", true))
			Expect(do.Data).To(HaveKeyWithValue("key5", map[string]interface{}{
				"x": map[string]interface{}{
					"y": "z",
				},
			}))
		})

		It("should throw an error if the imported key of a secret does not exist", func() {
			ctx := context.Background()
			defer ctx.Done()
			secret := &corev1.Secret{}
			secret.Name = "test-secret"
			secret.Namespace = "default"
			secret.Data = map[string][]byte{
				"key1": []byte("\"val1\""),
			}
			Expect(kubeClient.Create(ctx, secret)).To(Succeed())

			_, _, err := installations.GetDataImport(ctx, kubeClient, "", &installations.InstallationAndImports{}, lsv1alpha1.DataImport{
				Name: "imp",
				SecretRef: &lsv1alpha1.SecretReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      secret.Name,
						Namespace: secret.Namespace,
					},
					Key: "key2",
				},
			})
			Expect(err).To(HaveOccurred())
		})

	})

})
