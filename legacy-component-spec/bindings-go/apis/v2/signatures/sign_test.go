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
	"crypto/sha256"
	"fmt"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/signatures"
)

type TestSigner struct{}

func (s TestSigner) Sign(componentDescriptor cdv2.ComponentDescriptor, digest cdv2.DigestSpec) (*cdv2.SignatureSpec, error) {
	return &cdv2.SignatureSpec{
		Algorithm: "testSignAlgorithm",
		Value:     fmt.Sprintf("%s:%s-signed", digest.HashAlgorithm, digest.Value),
	}, nil
}

type TestVerifier struct{}

func (v TestVerifier) Verify(componentDescriptor cdv2.ComponentDescriptor, signature cdv2.Signature) error {
	if signature.Signature.Value != fmt.Sprintf("%s:%s-signed", signature.Digest.HashAlgorithm, signature.Digest.Value) {
		return fmt.Errorf("signature verification failed: Invalid signature")
	}
	return nil
}

type TestSHA256Hasher signatures.Hasher

var _ = ginkgo.Describe("Sign/Verify component-descriptor", func() {
	var baseCd cdv2.ComponentDescriptor
	testSHA256Hasher := signatures.Hasher{
		HashFunction:  sha256.New(),
		AlgorithmName: signatures.SHA256,
	}
	signatureName := "testSignatureName"
	correctBaseCdHash := "6c571bb6e351ae755baa7f26cbd1f600d2968ab8b88e25a3bab277e53afdc3ad"

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

	ginkgo.Describe("sign component-descriptor", func() {
		ginkgo.It("should add one signature", func() {
			err := signatures.SignComponentDescriptor(&baseCd, TestSigner{}, testSHA256Hasher, signatureName)
			Expect(err).To(BeNil())
			Expect(len(baseCd.Signatures)).To(BeIdenticalTo(1))
			Expect(baseCd.Signatures[0].Name).To(BeIdenticalTo(signatureName))
			Expect(baseCd.Signatures[0].Digest.NormalisationAlgorithm).To(BeIdenticalTo(string(cdv2.JsonNormalisationV1)))
			Expect(baseCd.Signatures[0].Digest.HashAlgorithm).To(BeIdenticalTo(signatures.SHA256))
			Expect(baseCd.Signatures[0].Digest.Value).To(BeIdenticalTo(correctBaseCdHash))
			Expect(baseCd.Signatures[0].Signature.Algorithm).To(BeIdenticalTo("testSignAlgorithm"))
			Expect(baseCd.Signatures[0].Signature.Value).To(BeIdenticalTo(fmt.Sprintf("%s:%s-signed", signatures.SHA256, correctBaseCdHash)))
		})
	})
	ginkgo.Describe("verify component-descriptor signature", func() {
		ginkgo.It("should verify one signature", func() {
			err := signatures.SignComponentDescriptor(&baseCd, TestSigner{}, testSHA256Hasher, signatureName)
			Expect(err).To(BeNil())
			Expect(len(baseCd.Signatures)).To(BeIdenticalTo(1))
			err = signatures.VerifySignedComponentDescriptor(&baseCd, TestVerifier{}, signatureName)
			Expect(err).To(BeNil())
		})
		ginkgo.It("should reject an invalid signature", func() {
			err := signatures.SignComponentDescriptor(&baseCd, TestSigner{}, testSHA256Hasher, signatureName)
			Expect(err).To(BeNil())
			Expect(len(baseCd.Signatures)).To(BeIdenticalTo(1))
			baseCd.Signatures[0].Signature.Value = "invalidSignature"
			err = signatures.VerifySignedComponentDescriptor(&baseCd, TestVerifier{}, signatureName)
			Expect(err).ToNot(BeNil())
		})
		ginkgo.It("should reject a missing signature", func() {
			err := signatures.VerifySignedComponentDescriptor(&baseCd, TestVerifier{}, signatureName)
			Expect(err).ToNot(BeNil())
		})

		ginkgo.It("should validate the correct signature if multiple are present", func() {
			err := signatures.SignComponentDescriptor(&baseCd, TestSigner{}, testSHA256Hasher, signatureName)
			Expect(err).To(BeNil())
			Expect(len(baseCd.Signatures)).To(BeIdenticalTo(1))

			baseCd.Signatures = append(baseCd.Signatures, cdv2.Signature{
				Name: "testSignAlgorithmNOTRight",
				Digest: cdv2.DigestSpec{
					HashAlgorithm:          "testAlgorithm",
					NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
					Value:                  "testValue",
				},
				Signature: cdv2.SignatureSpec{
					Algorithm: "testSigning",
					Value:     "AdditionalSignature",
				},
			})
			err = signatures.VerifySignedComponentDescriptor(&baseCd, TestVerifier{}, signatureName)
			Expect(err).To(BeNil())
		})
	})

	ginkgo.Describe("verify normalised component-descriptor digest with signed digest ", func() {
		ginkgo.It("should reject an invalid hash", func() {
			err := signatures.SignComponentDescriptor(&baseCd, TestSigner{}, testSHA256Hasher, signatureName)
			Expect(err).To(BeNil())
			Expect(len(baseCd.Signatures)).To(BeIdenticalTo(1))
			baseCd.Signatures[0].Digest.Value = "invalidHash"
			err = signatures.VerifySignedComponentDescriptor(&baseCd, TestVerifier{}, signatureName)
			Expect(err).ToNot(BeNil())
		})
		ginkgo.It("should reject a missing hash", func() {
			err := signatures.SignComponentDescriptor(&baseCd, TestSigner{}, testSHA256Hasher, signatureName)
			Expect(err).To(BeNil())
			Expect(len(baseCd.Signatures)).To(BeIdenticalTo(1))
			baseCd.Signatures[0].Digest = cdv2.DigestSpec{}
			err = signatures.VerifySignedComponentDescriptor(&baseCd, TestVerifier{}, signatureName)
			Expect(err).ToNot(BeNil())
		})
	})
})
