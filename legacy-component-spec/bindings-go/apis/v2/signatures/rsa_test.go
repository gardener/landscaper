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
	"encoding/hex"
	"encoding/pem"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/signatures"
)

var _ = Describe("RSA sign/verify", func() {
	var pathPrivateKey string
	var pathPublicKey string
	var stringToHashAndSign string
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = ioutil.TempDir("", "component-spec-test")
		Expect(err).To(BeNil())

		// openssl genpkey -out private.key -algorithm RSA
		pathPrivateKey = path.Join(dir, "private.key")
		createPrivateKeyCommand := exec.Command("openssl", "genpkey", "-out", pathPrivateKey, "-algorithm", "RSA")
		err = createPrivateKeyCommand.Run()
		Expect(err).To(BeNil())

		// openssl rsa -in private.key -outform PEM -pubout -out public.key
		pathPublicKey = path.Join(dir, "public.key")
		createPublicKeyCommand := exec.Command("openssl", "rsa", "-in", pathPrivateKey, "-outform", "PEM", "-pubout", "-out", pathPublicKey)
		err = createPublicKeyCommand.Run()
		Expect(err).To(BeNil())

		stringToHashAndSign = "TestStringToSign"
	})

	AfterEach(func() {
		os.RemoveAll(dir)
	})

	Describe("RSA sign with private key", func() {
		It("should create a signature", func() {
			hashOfString := sha256.Sum256([]byte(stringToHashAndSign))

			signer, err := signatures.CreateRSASignerFromKeyFile(pathPrivateKey, cdv2.MediaTypeRSASignature)
			Expect(err).To(BeNil())

			signature, err := signer.Sign(cdv2.ComponentDescriptor{}, cdv2.DigestSpec{
				HashAlgorithm:          signatures.SHA256,
				NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
				Value:                  hex.EncodeToString(hashOfString[:]),
			})
			Expect(err).To(BeNil())

			Expect(signature.MediaType).To(Equal(cdv2.MediaTypeRSASignature))
			Expect(signature.Algorithm).To(BeIdenticalTo(cdv2.RSAPKCS1v15))
			Expect(signature.Value).NotTo(BeNil())
		})

		It("should create a signature in pem format", func() {
			hashOfString := sha256.Sum256([]byte(stringToHashAndSign))

			signer, err := signatures.CreateRSASignerFromKeyFile(pathPrivateKey, cdv2.MediaTypePEM)
			Expect(err).To(BeNil())

			signature, err := signer.Sign(cdv2.ComponentDescriptor{}, cdv2.DigestSpec{
				HashAlgorithm:          signatures.SHA256,
				NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
				Value:                  hex.EncodeToString(hashOfString[:]),
			})
			Expect(err).To(BeNil())

			Expect(signature.MediaType).To(Equal(cdv2.MediaTypePEM))
			Expect(signature.Algorithm).To(BeIdenticalTo(cdv2.RSAPKCS1v15))
			Expect(signature.Value).NotTo(BeNil())

			pemBlock, rest := pem.Decode([]byte(signature.Value))
			Expect(pemBlock).ToNot(BeNil())
			Expect(len(rest)).To(BeZero())

			Expect(pemBlock.Type).To(Equal(cdv2.SignaturePEMBlockType))
			Expect(pemBlock.Headers[cdv2.SignatureAlgorithmHeader]).To(Equal(cdv2.RSAPKCS1v15))
			Expect(len(pemBlock.Bytes)).ToNot(BeZero())
		})
	})

	Describe("RSA sign and verify with public key", func() {
		It("should verify a signature", func() {
			hashOfString := sha256.Sum256([]byte(stringToHashAndSign))

			signer, err := signatures.CreateRSASignerFromKeyFile(pathPrivateKey, cdv2.MediaTypeRSASignature)
			Expect(err).To(BeNil())

			digest := cdv2.DigestSpec{
				HashAlgorithm:          signatures.SHA256,
				NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
				Value:                  hex.EncodeToString(hashOfString[:]),
			}
			signature, err := signer.Sign(cdv2.ComponentDescriptor{}, digest)
			Expect(err).To(BeNil())

			Expect(signature.Algorithm).To(BeIdenticalTo(cdv2.RSAPKCS1v15))
			Expect(signature.Value).NotTo(BeNil())

			verifier, err := signatures.CreateRSAVerifierFromKeyFile(pathPublicKey)
			Expect(err).To(BeNil())

			err = verifier.Verify(cdv2.ComponentDescriptor{}, cdv2.Signature{
				Digest:    digest,
				Signature: *signature,
			})
			Expect(err).To(BeNil())
		})

		It("should deny a signature from a wrong actor", func() {
			hashOfString := sha256.Sum256([]byte(stringToHashAndSign))

			//generate a wrong key (e.g. from a malicious actor)
			pathWrongPrivateKey := path.Join(dir, "privateWrong.key")
			createWrongPrivateKeyCommand := exec.Command("openssl", "genpkey", "-out", pathWrongPrivateKey, "-algorithm", "RSA")
			err := createWrongPrivateKeyCommand.Run()
			Expect(err).To(BeNil())

			signer, err := signatures.CreateRSASignerFromKeyFile(pathWrongPrivateKey, cdv2.MediaTypeRSASignature)
			Expect(err).To(BeNil())

			digest := cdv2.DigestSpec{
				HashAlgorithm:          signatures.SHA256,
				NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
				Value:                  hex.EncodeToString(hashOfString[:]),
			}
			signature, err := signer.Sign(cdv2.ComponentDescriptor{}, digest)
			Expect(err).To(BeNil())

			Expect(signature.Algorithm).To(BeIdenticalTo(cdv2.RSAPKCS1v15))
			Expect(signature.Value).NotTo(BeNil())

			verifier, err := signatures.CreateRSAVerifierFromKeyFile(pathPublicKey)
			Expect(err).To(BeNil())

			err = verifier.Verify(cdv2.ComponentDescriptor{}, cdv2.Signature{
				Digest:    digest,
				Signature: *signature,
			})
			Expect(err.Error()).To(BeIdenticalTo("unable to verify signature: crypto/rsa: verification error"))
		})
	})
})
