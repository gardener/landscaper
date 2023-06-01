// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package nodigest

import (
	"crypto"
	"hash"

	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/signing"
)

const Algorithm = metav1.NoDigest

func init() {
	signing.DefaultHandlerRegistry().RegisterHasher(Handler{})
}

// Handler is a signatures.Hasher compatible struct to hash with sha256.
type Handler struct{}

var _ signing.Hasher = Handler{}

func (h Handler) Algorithm() string {
	return Algorithm
}

// Create creates a Hasher instance for sha256.
func (_ Handler) Create() hash.Hash {
	return nil
}

func (_ Handler) Crypto() crypto.Hash {
	return 0
}
