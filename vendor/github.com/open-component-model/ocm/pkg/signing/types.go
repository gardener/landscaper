// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"crypto"
	"encoding/json"
	"hash"

	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
)

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
	// Sign returns the signature for the given digest
	Sign(cctx credentials.Context, digest string, hash crypto.Hash, issuer string, privatekey interface{}) (*Signature, error)
	// Algorithm is the name of the finally used signature algorithm.
	// A signer might be registered using a logical name, so there might
	// be multiple signer registration providing the same signature algorithm
	Algorithm() string
}

// Verifier interface is used to implement different verification algorithms.
// Each Verifier should have a matching Signer.
type Verifier interface {
	// Verify checks the signature, returns an error on verification failure
	Verify(digest string, hash crypto.Hash, sig *Signature, publickey interface{}) error
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
