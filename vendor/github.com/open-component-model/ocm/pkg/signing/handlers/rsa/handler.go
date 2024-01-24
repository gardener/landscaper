// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/signutils"
)

// Algorithm defines the type for the RSA PKCS #1 v1.5 signature algorithm.
const Algorithm = "RSASSA-PKCS1-V1_5"

// MediaType defines the media type for a plain RSA signature.
const MediaType = "application/vnd.ocm.signature.rsa"

// MediaTypePEM is used if the signature contains the public key certificate chain.
const MediaTypePEM = signutils.MediaTypePEM

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

func (h Handler) Sign(cctx credentials.Context, digest string, sctx signing.SigningContext) (signature *signing.Signature, err error) {
	privateKey, err := GetPrivateKey(sctx.GetPrivateKey())
	if err != nil {
		return nil, errors.Wrapf(err, "invalid rsa private key")
	}
	decodedHash, err := hex.DecodeString(digest)
	if err != nil {
		return nil, fmt.Errorf("failed decoding hash to bytes")
	}
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, sctx.GetHash(), decodedHash)
	if err != nil {
		return nil, fmt.Errorf("failed signing hash, %w", err)
	}

	media := MediaType
	value := hex.EncodeToString(sig)

	var iss string
	pub := sctx.GetPublicKey()
	if pub != nil {
		var pubKey signutils.GenericPublicKey
		certs, err := signutils.GetCertificateChain(pub, false)
		if err == nil && len(certs) > 0 {
			pubKey, _, err = GetPublicKey(certs[0].PublicKey)
			if err != nil {
				return nil, errors.ErrInvalidWrap(err, "public key")
			}
			err = signutils.VerifyCertificate(certs[0], certs[1:], sctx.GetRootCerts(), sctx.GetIssuer())
			if err != nil {
				return nil, errors.Wrapf(err, "public key certificate")
			}
			media = MediaTypePEM
			value = string(signutils.SignatureBytesToPem(Algorithm, sig, certs...))
			iss = certs[0].Subject.String()
		} else {
			pubKey, _, err = GetPublicKey(pub)
			if err != nil {
				return nil, errors.ErrInvalidWrap(err, "public key")
			}
		}
		if !privateKey.PublicKey.Equal(pubKey) {
			return nil, fmt.Errorf("invalid public key for private key")
		}
	}

	return &signing.Signature{
		Value:     value,
		MediaType: media,
		Algorithm: Algorithm,
		Issuer:    iss,
	}, nil
}

// Verify checks the signature, returns an error on verification failure.
func (h Handler) Verify(digest string, signature *signing.Signature, sctx signing.SigningContext) (err error) {
	var signatureBytes []byte

	publicKey, name, err := GetPublicKey(sctx.GetPublicKey())
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	switch signature.MediaType {
	case MediaType:
		signatureBytes, err = hex.DecodeString(signature.Value)
		if err != nil {
			return fmt.Errorf("unable to get signature value: failed decoding hash %s: %w", digest, err)
		}
	case signutils.MediaTypePEM:
		sig, algo, _, err := signutils.GetSignatureFromPem([]byte(signature.Value))
		if err != nil {
			return fmt.Errorf("unable to get signature from pem: %w", err)
		}
		if algo != "" && algo != Algorithm {
			return errors.ErrInvalid(signutils.KIND_SIGN_ALGORITHM, algo)
		}
		signatureBytes = sig
	default:
		return fmt.Errorf("invalid signature mediaType %s", signature.MediaType)
	}

	decodedHash, err := hex.DecodeString(digest)
	if err != nil {
		return fmt.Errorf("failed decoding hash %s: %w", digest, err)
	}

	if name != nil {
		if signature.Issuer != "" {
			iss, err := signutils.ParseDN(signature.Issuer)
			if err != nil {
				return errors.Wrapf(err, "signature issuer")
			}
			if signutils.MatchDN(*iss, *name) != nil {
				return fmt.Errorf("issuer %s does not match %s", signature.Issuer, name)
			}
		}
	}
	if err := rsa.VerifyPKCS1v15(publicKey, sctx.GetHash(), decodedHash, signatureBytes); err != nil {
		return fmt.Errorf("signature verification failed, %w", err)
	}

	return nil
}

func (_ Handler) CreateKeyPair() (priv signutils.GenericPublicKey, pub signutils.GenericPublicKey, err error) {
	return CreateKeyPair()
}

func CreateKeyPair() (priv signutils.GenericPublicKey, pub signutils.GenericPublicKey, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return key, &key.PublicKey, nil
}
