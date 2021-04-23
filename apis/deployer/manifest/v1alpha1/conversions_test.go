// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	lscore "github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/manifest"
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

			manifestConfig = &manifest.ProviderConfiguration{
				Kubeconfig:     "my-kubeconfig",
				UpdateStrategy: manifest.UpdateStrategyPatch,
				Manifests: []manifest.Manifest{
					{
						Policy: manifest.ManagePolicy,
						Manifest: &runtime.RawExtension{
							Raw: []byte("manifest1"),
						},
					},
					{
						Policy: manifest.ManagePolicy,
						Manifest: &runtime.RawExtension{
							Raw: []byte("manifest2"),
						},
					},
				},
			}
		)

		Context("v1alpha1 to manifest", func() {
			It("should convert all configuration and default the policy", func() {
				res := &manifest.ProviderConfiguration{}
				Expect(manifestv1alpha1.Convert_v1alpha1_ProviderConfiguration_To_manifest_ProviderConfiguration(v1alpha1Config, res, nil)).To(Succeed())
				Expect(res).To(Equal(manifestConfig))
			})
		})

		Context("manifest to v1alpha1", func() {
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

			manifestStatus = &manifest.ProviderStatus{
				ManagedResources: []manifest.ManagedResourceStatus{
					{
						Policy: manifest.ManagePolicy,
						Resource: lscore.TypedObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							ObjectReference: lscore.ObjectReference{
								Name:      "s1",
								Namespace: "default",
							},
						},
					},
					{
						Policy: manifest.ManagePolicy,
						Resource: lscore.TypedObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							ObjectReference: lscore.ObjectReference{
								Name:      "s2",
								Namespace: "default",
							},
						},
					},
				},
			}
		)

		It("v1alpha1 to manifest", func() {
			res := &manifest.ProviderStatus{}
			Expect(manifestv1alpha1.Convert_v1alpha1_ProviderStatus_To_manifest_ProviderStatus(v1alpha1Status, res, nil)).To(Succeed())
			Expect(res).To(Equal(manifestStatus))
		})

		It("manifest to v1alpha1", func() {
			res := &manifestv1alpha1.ProviderStatus{}
			Expect(manifestv1alpha1.Convert_manifest_ProviderStatus_To_v1alpha1_ProviderStatus(manifestStatus, res, nil)).To(Succeed())
			Expect(res).To(Equal(v1alpha1Status))
		})
	})

})
