// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blob

import (
	"fmt"
	"io"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/signing"
)

const GenericBlobDigestV1 = "genericBlobDigest/v1"

func init() {
	cpi.MustRegisterDigester(&defaultDigester{})
	cpi.SetDefaultDigester(&defaultDigester{})
}

type defaultDigester struct{}

var _ cpi.BlobDigester = (*defaultDigester)(nil)

func (d defaultDigester) GetType() cpi.DigesterType {
	return cpi.DigesterType{
		HashAlgorithm:          "",
		NormalizationAlgorithm: GenericBlobDigestV1,
	}
}

func (d defaultDigester) DetermineDigest(typ string, acc cpi.AccessMethod, preferred signing.Hasher) (*cpi.DigestDescriptor, error) {
	r, err := acc.Reader()
	if err != nil {
		return nil, err
	}
	hash := preferred.Create()

	if _, err := io.Copy(hash, r); err != nil {
		return nil, err
	}

	return &cpi.DigestDescriptor{
		Value:                  fmt.Sprintf("%x", hash.Sum(nil)),
		HashAlgorithm:          preferred.Algorithm(),
		NormalisationAlgorithm: GenericBlobDigestV1,
	}, nil
}
