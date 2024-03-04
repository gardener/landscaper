// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tsa

import (
	"crypto"
	"crypto/x509/pkix"
	"fmt"
	"time"

	cms "github.com/InfiniteLoopSpace/go_S-MIME/cms/protocol"
	"github.com/InfiniteLoopSpace/go_S-MIME/oid"
	tsa "github.com/InfiniteLoopSpace/go_S-MIME/timestamp"
	"github.com/go-test/deep"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing/signutils"
	"github.com/open-component-model/ocm/pkg/utils"
)

// NewMessageImprint creates a new MessageImprint using hash and digest.
func NewMessageImprint(hash crypto.Hash, digest []byte) (*MessageImprint, error) {
	digestAlgorithm := oid.HashToDigestAlgorithm[hash]
	if len(digestAlgorithm) == 0 {
		return nil, cms.ErrUnsupported
	}

	if !hash.Available() {
		return nil, cms.ErrUnsupported
	}

	if len(digest) != hash.Size() {
		return nil, cms.ASN1Error{"invalid hash size"}
	}

	return &MessageImprint{
		HashAlgorithm: pkix.AlgorithmIdentifier{Algorithm: digestAlgorithm},
		HashedMessage: digest,
	}, nil
}

func NewMessageImprintForData(data []byte, hash crypto.Hash) (*MessageImprint, error) {
	mi, err := tsa.NewMessageImprint(hash, data)
	if err != nil {
		return nil, err
	}
	return &mi, nil
}

func Request(url string, mi *tsa.MessageImprint) (*TimeStamp, time.Time, error) {
	if mi == nil {
		return nil, time.Time{}, fmt.Errorf("message imprint required")
	}
	req := tsa.TimeStampReq{
		Version:        1,
		CertReq:        true,
		Nonce:          tsa.GenerateNonce(),
		MessageImprint: *mi,
	}

	resp, err := req.Do(url)
	if err != nil {
		return nil, time.Time{}, errors.Wrapf(err, "requesting timestamp from %s", url)
	}

	sd, err := resp.TimeStampToken.SignedDataContent()
	if err != nil {
		return nil, time.Time{}, errors.Wrapf(err, "unexpected answer timestamp response from %s", url)
	}

	t, err := Verify(mi, sd, true)
	if err != nil {
		return nil, time.Time{}, errors.Wrapf(err, "cannot verify timestamp response")
	}
	return sd, *t, nil
}

func Verify(mi *tsa.MessageImprint, sd *TimeStamp, now bool, rootpool ...signutils.GenericCertificatePool) (*time.Time, error) {
	info, err := tsa.ParseInfo(sd.EncapContentInfo)
	if err != nil {
		return nil, err
	}
	if diff := deep.Equal(info.MessageImprint.HashAlgorithm.Algorithm, mi.HashAlgorithm.Algorithm); diff != nil {
		return nil, fmt.Errorf("hash algorithm mismatch: %s", diff)
	}
	if diff := deep.Equal(info.MessageImprint.HashedMessage, mi.HashedMessage); diff != nil {
		return nil, fmt.Errorf("digest mismatch: %s", diff)
	}
	opts := tsa.Opts
	if !now {
		opts.CurrentTime = info.GenTime
	}
	opts.Roots, err = signutils.GetCertPool(utils.Optional(rootpool...), false)
	if err != nil {
		return nil, errors.Wrapf(err, "root cert pool")
	}
	_, err = sd.Verify(opts, nil)
	if err != nil {
		return nil, err
	}
	return &info.GenTime, nil
}

func GetTimestamp(ts *TimeStamp) (time.Time, error) {
	info, err := tsa.ParseInfo(ts.EncapContentInfo)
	if err != nil {
		return time.Time{}, err
	}
	return info.GenTime, nil
}
