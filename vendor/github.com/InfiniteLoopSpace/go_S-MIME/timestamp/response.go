package timestamp

import (
	asn "github.com/InfiniteLoopSpace/go_S-MIME/asn1"
	cms "github.com/InfiniteLoopSpace/go_S-MIME/cms/protocol"
)

//TimeStampResp ::= SEQUENCE  {
//	status                  PKIStatusInfo,
//	timeStampToken          TimeStampToken     OPTIONAL  }
type TimeStampResp struct {
	Status         PKIStatusInfo
	TimeStampToken cms.ContentInfo `asn1:"optional"`
}

// ParseResponse parses a ASN.1 encoded TimeStampResp.
func ParseResponse(der []byte) (TimeStampResp, error) {
	var resp TimeStampResp

	rest, err := asn.Unmarshal(der, &resp)
	if err != nil {
		return resp, err
	}
	if len(rest) > 0 {
		return resp, cms.ErrTrailingData
	}

	return resp, nil
}

// Info returns the timestampinfo from a response.
func (r TimeStampResp) Info() (TSTInfo, error) {
	var nilInfo TSTInfo

	if err := r.Status.GetError(); err != nil {
		return nilInfo, err
	}

	sd, err := r.TimeStampToken.SignedDataContent()
	if err != nil {
		return nilInfo, err
	}

	return ParseInfo(sd.EncapContentInfo)
}
