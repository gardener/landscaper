// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package sha512

import (
	"crypto"
	"crypto/sha512"
	"hash"

	"github.com/open-component-model/ocm/pkg/signing"
)

var Algorithm = crypto.SHA512.String()

func init() {
	signing.DefaultHandlerRegistry().RegisterHasher(Handler{})
}

// Handler is a signatures.Hasher compatible struct to hash with sha256.
type Handler struct{}

var _ signing.Hasher = Handler{}

func (_ Handler) Algorithm() string {
	return Algorithm
}

// Create creates a Hasher instance for no digest.
func (_ Handler) Create() hash.Hash {
	return sha512.New()
}

func (_ Handler) Crypto() crypto.Hash {
	return crypto.SHA512
}
