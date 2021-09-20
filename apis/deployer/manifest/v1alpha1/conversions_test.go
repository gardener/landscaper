// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestcore "github.com/gardener/landscaper/apis/deployer/manifest"
	manifestv1alpha1 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha1"
)

var _ = Describe("Conversion", func() {

	Context("ProviderConfiguration", func() {

		var (
			v1alpha1Config = &manifestv1alpha1.ProviderConfiguration{
				Kubeconfig:     "my-kubeconfig",
				UpdateStrategy: manifestv1alpha1.UpdateStrategyPatch,
				Manifests: []*runtime.RawExtension{
					{
						Raw: []byte("manifest1"),
					},
					{
						Raw: []byte("manifest2"),
					},
				},
			}

			manifestConfig = &manifestcore.ProviderConfiguration{
				Kubeconfig:     "my-kubeconfig",
				UpdateStrategy: manifestcore.UpdateStrategyPatch,
				Manifests: []managedresource.Manifest{
					{
						Policy: managedresource.ManagePolicy,
						Manifest: &runtime.RawExtension{
							Raw: []byte("manifest1"),
						},
					},
					{
						Policy: managedresource.ManagePolicy,
						Manifest: &runtime.RawExtension{
							Raw: []byte("manifest2"),
						},
					},
				},
			}
		)

		Context("v1alpha1 to manifestcore", func() {
			It("should convert all configuration and default the policy", func() {
				res := &manifestcore.ProviderConfiguration{}
				Expect(manifestv1alpha1.Convert_v1alpha1_ProviderConfiguration_To_manifest_ProviderConfiguration(v1alpha1Config, res, nil)).To(Succeed())
				Expect(res).To(Equal(manifestConfig))
			})
		})

		Context("manifestcore to v1alpha1", func() {
			It("should convert all configuration and default the policy", func() {
				res := &manifestv1alpha1.ProviderConfiguration{}
				Expect(manifestv1alpha1.Convert_manifest_ProviderConfiguration_To_v1alpha1_ProviderConfiguration(manifestConfig, res, nil)).To(Succeed())
				Expect(res).To(Equal(v1alpha1Config))
			})
		})
	})

	Context("ProviderStatus", func() {

		var (
			v1alpha1Status = &manifestv1alpha1.ProviderStatus{
				ManagedResources: []lsv1alpha1.TypedObjectReference{
					{
						APIVersion: "v1",
						Kind:       "Secret",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      "s1",
							Namespace: "default",
						},
					},
					{
						APIVersion: "v1",
						Kind:       "Secret",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      "s2",
							Namespace: "default",
						},
					},
				},
			}

			manifestStatus = &manifestcore.ProviderStatus{
				ManagedResources: []managedresource.ManagedResourceStatus{
					{
						Policy: managedresource.ManagePolicy,
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "s1",
							Namespace:  "default",
						},
					},
					{
						Policy: managedresource.ManagePolicy,
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "s2",
							Namespace:  "default",
						},
					},
				},
			}
		)

		It("v1alpha1 to manifestcore", func() {
			res := &manifestcore.ProviderStatus{}
			Expect(manifestv1alpha1.Convert_v1alpha1_ProviderStatus_To_manifest_ProviderStatus(v1alpha1Status, res, nil)).To(Succeed())
			Expect(res).To(Equal(manifestStatus))
		})

		It("manifestcore to v1alpha1", func() {
			res := &manifestv1alpha1.ProviderStatus{}
			Expect(manifestv1alpha1.Convert_manifest_ProviderStatus_To_v1alpha1_ProviderStatus(manifestStatus, res, nil)).To(Succeed())
			Expect(res).To(Equal(v1alpha1Status))
		})
	})

})
