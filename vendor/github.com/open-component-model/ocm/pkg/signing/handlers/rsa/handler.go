// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package rsa

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing"
)

// Algorithm defines the type for the RSA PKCS #1 v1.5 signature algorithm.
const Algorithm = "RSASSA-PKCS1-V1_5"

// MediaType defines the media type for a plain RSA signature.
const MediaType = "application/vnd.ocm.signature.rsa"

// MediaTypePEM defines the media type for PEM formatted data.
const MediaTypePEM = "application/x-pem-file"

// SignaturePEMBlockType defines the type of a signature pem block.
const SignaturePEMBlockType = "SIGNATURE"

// SignaturePEMBlockAlgorithmHeader defines the header in a signature pem block where the signature algorithm is defined.
const SignaturePEMBlockAlgorithmHeader = "Signature Algorithm"

func init() {
	signing.DefaultHandlerRegistry().RegisterSigner(Algorithm, Handler{})
}

type (
	PrivateKey = rsa.PrivateKey
	PublicKey  = rsa.PublicKey
)

// Handler is a signatures.Signer compatible struct to sign with RSASSA-PKCS1-V1_5.
// and a signatures.Verifier compatible struct to verify RSASSA-PKCS1-V1_5 signatures.
type Handler struct{}

var _ Handler = Handler{}

func (h Handler) Algorithm() string {
	return Algorithm
}

func (h Handler) Sign(cctx credentials.Context, digest string, hash crypto.Hash, issuer string, key interface{}) (signature *signing.Signature, err error) {
	privateKey, err := GetPrivateKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid rsa private key")
	}
	decodedHash, err := hex.DecodeString(digest)
	if err != nil {
		return nil, fmt.Errorf("failed decoding hash to bytes")
	}
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, hash, decodedHash)
	if err != nil {
		return nil, fmt.Errorf("failed signing hash, %w", err)
	}
	return &signing.Signature{
		Value:     hex.EncodeToString(sig),
		MediaType: MediaType,
		Algorithm: Algorithm,
		Issuer:    issuer,
	}, nil
}

// Verify checks the signature, returns an error on verification failure.
func (h Handler) Verify(digest string, hash crypto.Hash, signature *signing.Signature, key interface{}) (err error) {
	var signatureBytes []byte

	publicKey, names, err := GetPublicKey(key)
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	switch signature.MediaType {
	case MediaType:
		signatureBytes, err = hex.DecodeString(signature.Value)
		if err != nil {
			return fmt.Errorf("unable to get signature value: failed decoding hash %s: %w", digest, err)
		}
	case MediaTypePEM:
		signaturePemBlocks, err := GetSignaturePEMBlocks([]byte(signature.Value))
		if err != nil {
			return fmt.Errorf("unable to get signature pem blocks: %w", err)
		}
		if len(signaturePemBlocks) != 1 {
			return fmt.Errorf("expected 1 signature pem block, found %d", len(signaturePemBlocks))
		}
		signatureBytes = signaturePemBlocks[0].Bytes
	default:
		return fmt.Errorf("invalid signature mediaType %s", signature.MediaType)
	}

	decodedHash, err := hex.DecodeString(digest)
	if err != nil {
		return fmt.Errorf("failed decoding hash %s: %w", digest, err)
	}

	if names != nil {
		if signature.Issuer != "" {
			found := false

			for _, n := range names {
				if n == signature.Issuer {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("issuer %q does not match %v", signature.Issuer, names)
			}
		}
	}
	if err := rsa.VerifyPKCS1v15(publicKey, hash, decodedHash, signatureBytes); err != nil {
		return fmt.Errorf("signature verification failed, %w", err)
	}

	return nil
}

// GetSignaturePEMBlocks returns all signature pem blocks from a list of pem blocks.
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

		if currentBlock.Type == SignaturePEMBlockType {
			signatureBlocks = append(signatureBlocks, currentBlock)
		}

		if len(pemData) == 0 {
			break
		}
	}

	return signatureBlocks, nil
}

func (_ Handler) CreateKeyPair() (priv interface{}, pub interface{}, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return key, &key.PublicKey, nil
}
