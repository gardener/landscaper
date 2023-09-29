// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"encoding/hex"
	"hash"

	"github.com/open-component-model/ocm/pkg/errors"
)

func Hash(hash hash.Hash, data []byte) (string, error) {
	hash.Reset()
	if _, err := hash.Write(data); err != nil {
		return "", errors.Wrapf(err, "failed hashing")
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
