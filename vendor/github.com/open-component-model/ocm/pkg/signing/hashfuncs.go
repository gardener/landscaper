// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"crypto"
	"strings"
)

var hashfuncs = map[string]crypto.Hash{
	"sha256": crypto.SHA256,
	"sha512": crypto.SHA512,
}

func init() {
	for h := crypto.Hash(1); h < 1000; h++ {
		s := h.String()
		if strings.HasPrefix(s, "unknown") {
			break
		}
		hashfuncs[s] = h
	}
}

func NormalizeHashAlgorithm(algo string) string {
	h := hashfuncs[algo]
	if h != 0 {
		return h.String()
	}
	return algo
}
