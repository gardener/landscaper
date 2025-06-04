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

package signatures

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	cdv2 "github.com/gardener/landscaper/component-spec-bindings-go/apis/v2"
)

// RSASigner is a signatures.Signer compatible struct to sign with RSASSA-PKCS1-V1_5.
type RSASigner struct {
	privateKey rsa.PrivateKey
	mediaType  string
}

// CreateRSASignerFromKeyFile creates an Instance of RSASigner with the given private key.
// The private key has to be in the PKCS #1, ASN.1 DER form, see x509.ParsePKCS1PrivateKey.
// mediaType defines the format of the signature that is saved to the component descriptor.
func CreateRSASignerFromKeyFile(pathToPrivateKey, mediaType string) (*RSASigner, error) {
	privKeyFile, err := os.ReadFile(pathToPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to open private key file: %w", err)
	}

	block, _ := pem.Decode([]byte(privKeyFile))
	if block == nil {
		return nil, fmt.Errorf("unable to decode pem formatted block in key: %w", err)
	}
	untypedPrivateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	key, ok := untypedPrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("parsed private key is not of type *rsa.PrivateKey: %T", untypedPrivateKey)
	}

	return &RSASigner{
		privateKey: *key,
		mediaType:  mediaType,
	}, nil
}

// Sign returns the signature for the data for the component descriptor.
func (s RSASigner) Sign(componentDescriptor cdv2.ComponentDescriptor, digest cdv2.DigestSpec) (*cdv2.SignatureSpec, error) {
	hashfunc, ok := HashFunctions[digest.HashAlgorithm]
	if !ok {
		return nil, fmt.Errorf("unknown hash algorithm %s", digest.HashAlgorithm)
	}

	decodedHash, err := hex.DecodeString(digest.Value)
	if err != nil {
		return nil, fmt.Errorf("unable to hex decode hash: %w", err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, &s.privateKey, hashfunc, decodedHash)
	if err != nil {
		return nil, fmt.Errorf("unable to sign hash: %w", err)
	}

	switch s.mediaType {
	case cdv2.MediaTypeRSASignature:
		return &cdv2.SignatureSpec{
			Algorithm: cdv2.RSAPKCS1v15,
			Value:     hex.EncodeToString(signature),
			MediaType: cdv2.MediaTypeRSASignature,
		}, nil
	case cdv2.MediaTypePEM:
		signatureBlock := &pem.Block{
			Type: cdv2.SignaturePEMBlockType,
			Headers: map[string]string{
				cdv2.SignatureAlgorithmHeader: cdv2.RSAPKCS1v15,
			},
			Bytes: signature,
		}

		buf := bytes.NewBuffer([]byte{})
		if err := pem.Encode(buf, signatureBlock); err != nil {
			return nil, fmt.Errorf("unable to encode signature pem block: %w", err)
		}
		return &cdv2.SignatureSpec{
			Algorithm: cdv2.RSAPKCS1v15,
			Value:     buf.String(),
			MediaType: cdv2.MediaTypePEM,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported signature media type %s", s.mediaType)
	}
}

// RSAVerifier is a signatures.Verifier compatible struct to verify RSASSA-PKCS1-V1_5 signatures.
type RSAVerifier struct {
	publicKey rsa.PublicKey
}

// CreateRSAVerifier creates an instance of RsaVerifier from a given rsa public key.
func CreateRSAVerifier(publicKey *rsa.PublicKey) (*RSAVerifier, error) {
	if publicKey == nil {
		return nil, errors.New("public key must not be nil")
	}

	verifier := RSAVerifier{
		publicKey: *publicKey,
	}

	return &verifier, nil
}

// CreateRSAVerifierFromKeyFile creates an instance of RsaVerifier from a rsa public key file.
// The private key has to be in the PKIX, ASN.1 DER form, see x509.ParsePKIXPublicKey.
func CreateRSAVerifierFromKeyFile(pathToPublicKey string) (*RSAVerifier, error) {
	publicKey, err := os.ReadFile(pathToPublicKey)
	if err != nil {
		return nil, fmt.Errorf("unable to open public key file: %w", err)
	}
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return nil, fmt.Errorf("unable to decode pem formatted block in key: %w", err)
	}
	untypedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key: %w", err)
	}
	switch key := untypedKey.(type) {
	case *rsa.PublicKey:
		return CreateRSAVerifier(key)
	default:
		return nil, fmt.Errorf("parsed public key is not of type *rsa.PublicKey: %T", key)
	}
}

// Verify checks the signature, returns an error on verification failure
func (v RSAVerifier) Verify(componentDescriptor cdv2.ComponentDescriptor, signature cdv2.Signature) error {
	var signatureBytes []byte
	var err error
	switch signature.Signature.MediaType {
	case cdv2.MediaTypeRSASignature:
		signatureBytes, err = hex.DecodeString(signature.Signature.Value)
		if err != nil {
			return fmt.Errorf("unable to hex decode signature %s: %w", signature.Signature.Value, err)
		}
	case cdv2.MediaTypePEM:
		signaturePemBlocks, err := GetSignaturePEMBlocks([]byte(signature.Signature.Value))
		if err != nil {
			return fmt.Errorf("unable to get signature pem blocks: %w", err)
		}
		if len(signaturePemBlocks) != 1 {
			return fmt.Errorf("expected 1 signature pem block, found %d", len(signaturePemBlocks))
		}
		signatureBytes = signaturePemBlocks[0].Bytes
	default:
		return fmt.Errorf("invalid signature mediaType %s", signature.Signature.MediaType)
	}

	hashfunc, ok := HashFunctions[signature.Digest.HashAlgorithm]
	if !ok {
		return fmt.Errorf("unknown hash algorithm %s", signature.Digest.HashAlgorithm)
	}

	decodedHash, err := hex.DecodeString(signature.Digest.Value)
	if err != nil {
		return fmt.Errorf("unable to hex decode hash %s: %w", signature.Digest.Value, err)
	}

	if err := rsa.VerifyPKCS1v15(&v.publicKey, hashfunc, decodedHash, signatureBytes); err != nil {
		return fmt.Errorf("unable to verify signature: %w", err)
	}

	return nil
}

// GetSignaturePEMBlocks returns all signature pem blocks from a list of pem blocks
func GetSignaturePEMBlocks(pemData []byte) ([]*pem.Block, error) {
	if len(pemData) == 0 {
		return []*pem.Block{}, nil
	}

	signatureBlocks := []*pem.Block{}
	for {
		var currentBlock *pem.Block
		currentBlock, pemData = pem.Decode(pemData)
		if currentBlock == nil && len(pemData) > 0 {
			return nil, fmt.Errorf("unable to decode pem block %s", string(pemData))
		}

		if currentBlock.Type == cdv2.SignaturePEMBlockType {
			signatureBlocks = append(signatureBlocks, currentBlock)
		}

		if len(pemData) == 0 {
			break
		}
	}

	return signatureBlocks, nil
}
