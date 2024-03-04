// Package timestamp implements the timestamp protocol rfc 3161
package timestamp

import (
	"crypto"
	"crypto/x509"
	"time"

	asn1 "github.com/InfiniteLoopSpace/go_S-MIME/asn1"
	cms "github.com/InfiniteLoopSpace/go_S-MIME/cms/protocol"
	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

const (
	contentTypeTSQuery = "application/timestamp-query"
	contentTypeTSReply = "application/timestamp-reply"
	nonceBytes         = 16
)

var (
	//Opts are options for timestamp certificate verficiation.
	Opts = x509.VerifyOptions{
		Intermediates: x509.NewCertPool(),
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}
)

// FetchTSToken tries to fetch a TSTokem of the given msg with hash using the given URL.
func FetchTSToken(url string, msg []byte, hash crypto.Hash) (tsToken cms.ContentInfo, err error) {
	req, err := newTSRequest(msg, hash)
	if err != nil {
		return
	}

	resp, err := req.Do(url)
	if err != nil {
		return
	}

	if err = resp.Status.GetError(); err != nil {
		return
	}

	sd, err := resp.TimeStampToken.SignedDataContent()

	if err != nil {
		return
	}

	_, err = sd.Verify(Opts, nil)

	return resp.TimeStampToken, err
}

// VerfiyTS verfies the given TSToken and returns the TSTInfo.
func VerfiyTS(ci cms.ContentInfo) (info TSTInfo, err error) {
	if !ci.ContentType.Equal(oid.SignedData) {
		err = cms.ErrUnsupported
		return
	}

	sd := cms.SignedData{}

	_, err = asn1.Unmarshal(ci.Content.Bytes, &sd)
	if err != nil {
		return
	}

	_, err = sd.Verify(Opts, nil)
	if err != nil {
		return
	}

	info, err = ParseInfo(sd.EncapContentInfo)

	return
}
