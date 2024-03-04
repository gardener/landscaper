// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tsa

import (
	"encoding/pem"

	"github.com/InfiniteLoopSpace/go_S-MIME/asn1"
	cms "github.com/InfiniteLoopSpace/go_S-MIME/cms/protocol"

	"github.com/open-component-model/ocm/pkg/errors"
)

const PRM_BLOCK_TYPE = "TIMESTAMP INFO"

func ToPem(sd *TimeStamp) ([]byte, error) {
	data, err := asn1.Marshal(*sd)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot marshal timestamp data")
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:    PRM_BLOCK_TYPE,
		Headers: nil,
		Bytes:   data,
	})
	return pemBytes, nil
}

func FromPem(data []byte) (*TimeStamp, error) {
	block, rest := pem.Decode(data)
	if block == nil || len(rest) > 0 {
		return nil, errors.ErrInvalid("timestamp")
	}
	if block.Type != PRM_BLOCK_TYPE {
		return nil, errors.ErrInvalid("PEM block type", block.Type, "timestamp")
	}
	var n cms.SignedData
	_, err := asn1.Unmarshal(block.Bytes, &n)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
