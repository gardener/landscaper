// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"crypto"
	"crypto/x509/pkix"
	"encoding/json"
	"hash"

	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/signing/signutils"
)

type SigningContext interface {
	GetHash() crypto.Hash
	GetPrivateKey() signutils.GenericPrivateKey
	GetPublicKey() signutils.GenericPublicKey
	GetRootCerts() signutils.GenericCertificatePool
	GetIssuer() *pkix.Name
}

type DefaultSigningContext struct {
	Hash       crypto.Hash
	PrivateKey signutils.GenericPrivateKey
	PublicKey  signutils.GenericPublicKey
	RootCerts  signutils.GenericCertificatePool
	Issuer     *pkix.Name
}

var _ SigningContext = (*DefaultSigningContext)(nil)

func (d *DefaultSigningContext) GetHash() crypto.Hash {
	return d.Hash
}

func (d *DefaultSigningContext) GetPrivateKey() signutils.GenericPrivateKey {
	return d.PrivateKey
}

func (d *DefaultSigningContext) GetPublicKey() signutils.GenericPublicKey {
	return d.PublicKey
}

func (d *DefaultSigningContext) GetRootCerts() signutils.GenericCertificatePool {
	return d.RootCerts
}

func (d *DefaultSigningContext) GetIssuer() *pkix.Name {
	return d.Issuer
}

type Signature struct { //nolint: musttag // only for string output
	Value     string
	MediaType string
	Algorithm string
	Issuer    string
}

func (s *Signature) String() string {
	data, err := json.Marshal(s)
	if err != nil {
		logrus.Error(err)
	}

	return string(data)
}

// Signer interface is used to implement different signing algorithms.
// Each Signer should have a matching Verifier.
type Signer interface {
	// Sign returns the signature for the given digest.
	// If known a given public key can be passed. The signer may
	// decide to put a trusted public key into the signature,
	// for example for public keys provided by organization validated
	// certificates.
	// If used the key and/or certificate must be validated, for certificates
	// the distinguished name must match the issuer.
	Sign(cctx credentials.Context, digest string, sctx SigningContext) (*Signature, error)
	// Algorithm is the name of the finally used signature algorithm.
	// A signer might be registered using a logical name, so there might
	// be multiple signer registration providing the same signature algorithm
	Algorithm() string
}

// Verifier interface is used to implement different verification algorithms.
// Each Verifier should have a matching Signer.
type Verifier interface {
	// Verify checks the signature, returns an error on verification failure
	Verify(digest string, sig *Signature, sctx SigningContext) error
	Algorithm() string
}

// SignatureHandler can create and verify signature of a dedicated type.
type SignatureHandler interface {
	Algorithm() string
	Signer
	Verifier
}

// Hasher creates a new hash.Hash interface.
type Hasher interface {
	Algorithm() string
	Create() hash.Hash
	Crypto() crypto.Hash
}
