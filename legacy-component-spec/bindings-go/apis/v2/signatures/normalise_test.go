// Copyright 2022 Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package signatures_test

import (
	"context"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/signatures"
)

var _ = ginkgo.Describe("Normalise/Hash component-descriptor", func() {
	var baseCd cdv2.ComponentDescriptor
	correctBaseCdHash := "6c571bb6e351ae755baa7f26cbd1f600d2968ab8b88e25a3bab277e53afdc3ad"
	//corresponding normalised CD:
	//[{"component":[{"componentReferences":[[{"componentName":"compRefNameComponentName"},{"digest":[{"hashAlgorithm":"sha256"},{"normalisationAlgorithm":"jsonNormalisation/v1"},{"value":"00000000000000"}]},{"extraIdentity":[{"refKey":"refName"}]},{"name":"compRefName"},{"version":"v0.0.2compRef"}]]},{"name":"CD-Name"},{"provider":""},{"resources":[[{"digest":[{"hashAlgorithm":"sha256"},{"normalisationAlgorithm":"ociArtifactDigest/v1"},{"value":"00000000000000"}]},{"extraIdentity":[{"key":"value"}]},{"name":"Resource1"},{"relation": ""},{"type",""},{"version":"v0.0.3resource"}]]},{"version":"v0.0.1"}]},{"meta":[{"schemaVersion":"v2"}]}]
	ginkgo.BeforeEach(func() {
		baseCd = cdv2.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: "v2",
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "CD-Name",
					Version: "v0.0.1",
				},
				ComponentReferences: []cdv2.ComponentReference{
					{
						Name:          "compRefName",
						ComponentName: "compRefNameComponentName",
						Version:       "v0.0.2compRef",
						ExtraIdentity: cdv2.Identity{
							"refKey": "refName",
						},
						Digest: &cdv2.DigestSpec{
							HashAlgorithm:          signatures.SHA256,
							NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
							Value:                  "00000000000000",
						},
					},
				},
				Resources: []cdv2.Resource{
					{
						IdentityObjectMeta: cdv2.IdentityObjectMeta{
							Name:    "Resource1",
							Version: "v0.0.3resource",
							ExtraIdentity: cdv2.Identity{
								"key": "value",
							},
						},
						Digest: &cdv2.DigestSpec{
							HashAlgorithm:          signatures.SHA256,
							NormalisationAlgorithm: string(cdv2.OciArtifactDigestV1),
							Value:                  "00000000000000",
						},
						Access: cdv2.NewUnstructuredType(cdv2.OCIRegistryType, map[string]interface{}{"imageRef": "ref"}),
					},
				},
			},
		}
	})

	ginkgo.Describe("missing componentReference Digest", func() {
		ginkgo.It("should fail to hash", func() {
			baseCd.ComponentSpec.ComponentReferences[0].Digest = nil
			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			hash, err := signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(hash).To(BeNil())
			Expect(err).ToNot(BeNil())
		})
	})
	ginkgo.Describe("should give the correct hash", func() {
		ginkgo.It("with sha256", func() {
			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			hash, err := signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())
			Expect(hash.Value).To(Equal(correctBaseCdHash))
		})
	})
	ginkgo.Describe("should ignore modifications in unhashed fields", func() {
		ginkgo.It("should succeed with signature changes", func() {
			baseCd.Signatures = append(baseCd.Signatures, cdv2.Signature{
				Name: "TestSig",
				Digest: cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
					Value:                  "00000",
				},
				Signature: cdv2.SignatureSpec{
					Algorithm: "test",
					Value:     "0000",
				},
			})
			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			hash, err := signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())
			Expect(hash.Value).To(Equal(correctBaseCdHash))
		})
		ginkgo.It("should succeed with source changes", func() {
			baseCd.Sources = append(baseCd.Sources, cdv2.Source{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "source1",
					Version: "v0.0.0",
				},
			})
			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			hash, err := signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())
			Expect(hash.Value).To(Equal(correctBaseCdHash))
		})
		ginkgo.It("should succeed with resource access reference changes", func() {
			access, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("ociRef/path/to/image"))
			Expect(err).To(BeNil())
			baseCd.Resources[0].Access = &access
			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			hash, err := signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())
			Expect(hash.Value).To(Equal(correctBaseCdHash))
		})

	})
	ginkgo.Describe("should correctly handle empty access and digest", func() {
		ginkgo.It("should be equal hash for access.type == None and access == nil", func() {
			baseCd.Resources[0].Access = nil
			baseCd.Resources[0].Digest = nil

			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			hash, err := signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())

			//add access to resource
			access := cdv2.NewEmptyUnstructured("None")
			Expect(err).To(BeNil())
			baseCd.Resources[0].Access = access
			hash2, err := signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())
			Expect(hash).To(Equal(hash2))
		})
		ginkgo.It("should fail if digest is empty", func() {
			baseCd.Resources[0].Digest = nil

			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			_, err = signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(HaveOccurred())
		})
		ginkgo.It("should succed if digest is empty and access is nil", func() {
			baseCd.Resources[0].Access = nil
			baseCd.Resources[0].Digest = nil

			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			_, err = signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())
		})
		ginkgo.It("should fail if first is nil access and an access is added but a digest is missing", func() {
			baseCd.Resources[0].Access = nil
			baseCd.Resources[0].Digest = nil

			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			_, err = signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())

			//add ociRegistryAccess
			access, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("ociRef/path/to/image"))
			Expect(err).To(BeNil())
			baseCd.Resources[0].Access = &access
			_, err = signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(HaveOccurred())
		})
		ginkgo.It("should fail if first is none access.type and an access is added but a digest is missing", func() {
			baseCd.Resources[0].Access = cdv2.NewEmptyUnstructured("None")
			baseCd.Resources[0].Digest = nil

			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			_, err = signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(BeNil())

			//add ociRegistryAccess
			access, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("ociRef/path/to/image"))
			Expect(err).To(BeNil())
			baseCd.Resources[0].Access = &access
			_, err = signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(HaveOccurred())
		})
		ginkgo.It("should fail if access is nil and digest is set", func() {
			baseCd.Resources[0].Access = nil

			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			_, err = signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(HaveOccurred())
		})
		ginkgo.It("should fail if access.type is None and digest is set", func() {
			baseCd.Resources[0].Access = cdv2.NewEmptyUnstructured("None")

			hasher, err := signatures.HasherForName(signatures.SHA256)
			Expect(err).To(BeNil())
			_, err = signatures.HashForComponentDescriptor(baseCd, *hasher)
			Expect(err).To(HaveOccurred())
		})
	})
	ginkgo.Describe("add digest to cd", func() {
		ginkgo.It("should succed if existing digest match calculated", func() {
			err := signatures.AddDigestsToComponentDescriptor(context.TODO(), &baseCd, func(ctx context.Context, cd cdv2.ComponentDescriptor, cr cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
					Value:                  "00000000000000",
				}, nil
			}, func(ctx context.Context, cd cdv2.ComponentDescriptor, r cdv2.Resource) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.OciArtifactDigestV1),
					Value:                  "00000000000000",
				}, nil
			})
			Expect(err).To(BeNil())
		})
		ginkgo.It("should fail if calcuated componentReference digest is different", func() {
			err := signatures.AddDigestsToComponentDescriptor(context.TODO(), &baseCd, func(ctx context.Context, cd cdv2.ComponentDescriptor, cr cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
					Value:                  "00000000000000-different",
				}, nil
			}, func(ctx context.Context, cd cdv2.ComponentDescriptor, r cdv2.Resource) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.OciArtifactDigestV1),
					Value:                  "00000000000000",
				}, nil
			})
			Expect(err).To(HaveOccurred())
		})
		ginkgo.It("should fail if calcuated resource digest is different", func() {
			err := signatures.AddDigestsToComponentDescriptor(context.TODO(), &baseCd, func(ctx context.Context, cd cdv2.ComponentDescriptor, cr cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
					Value:                  "00000000000000",
				}, nil
			}, func(ctx context.Context, cd cdv2.ComponentDescriptor, r cdv2.Resource) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.OciArtifactDigestV1),
					Value:                  "00000000000000-different",
				}, nil
			})
			Expect(err).To(HaveOccurred())
		})
		ginkgo.It("should add digest if missing", func() {
			baseCd.ComponentReferences[0].Digest = nil
			baseCd.Resources[0].Digest = nil

			err := signatures.AddDigestsToComponentDescriptor(context.TODO(), &baseCd, func(ctx context.Context, cd cdv2.ComponentDescriptor, cr cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
					Value:                  "00000000000000",
				}, nil
			}, func(ctx context.Context, cd cdv2.ComponentDescriptor, r cdv2.Resource) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.OciArtifactDigestV1),
					Value:                  "00000000000000",
				}, nil
			})
			Expect(err).To(BeNil())

			Expect(baseCd.ComponentReferences[0].Digest).To(Equal(&cdv2.DigestSpec{
				HashAlgorithm:          signatures.SHA256,
				NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
				Value:                  "00000000000000",
			}))
			Expect(baseCd.Resources[0].Digest).To(Equal(&cdv2.DigestSpec{
				HashAlgorithm:          signatures.SHA256,
				NormalisationAlgorithm: string(cdv2.OciArtifactDigestV1),
				Value:                  "00000000000000",
			}))
		})
		ginkgo.It("should preserve the EXCLUDE-FROM-SIGNATURE digest", func() {
			baseCd.Resources[0].Digest = cdv2.NewExcludeFromSignatureDigest()

			err := signatures.AddDigestsToComponentDescriptor(context.TODO(), &baseCd, func(ctx context.Context, cd cdv2.ComponentDescriptor, cr cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
					Value:                  "00000000000000",
				}, nil
			}, func(ctx context.Context, cd cdv2.ComponentDescriptor, r cdv2.Resource) (*cdv2.DigestSpec, error) {
				return &cdv2.DigestSpec{
					HashAlgorithm:          signatures.SHA256,
					NormalisationAlgorithm: string(cdv2.OciArtifactDigestV1),
					Value:                  "00000000000000",
				}, nil
			})
			Expect(err).To(BeNil())

			Expect(baseCd.ComponentReferences[0].Digest).To(Equal(&cdv2.DigestSpec{
				HashAlgorithm:          signatures.SHA256,
				NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
				Value:                  "00000000000000",
			}))
			Expect(baseCd.Resources[0].Digest).To(Equal(cdv2.NewExcludeFromSignatureDigest()))
		})
	})
})
