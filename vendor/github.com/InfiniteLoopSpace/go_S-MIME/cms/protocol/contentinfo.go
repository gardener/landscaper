// Package protocol implemets parts of cryptographic message syntax RFC 5652.
// This package is mostly for handling of the asn1 sturctures of cms. For
// de/encryption and signing/verfiying use to package cms.
package protocol

import (
	"encoding/asn1"
	"fmt"

	asn "github.com/InfiniteLoopSpace/go_S-MIME/asn1"
	"github.com/InfiniteLoopSpace/go_S-MIME/b64"
)

// ContentInfo ::= SEQUENCE {
//   contentType ContentType,
//   content [0] EXPLICIT ANY DEFINED BY contentType }
//
// ContentType ::= OBJECT IDENTIFIER
type ContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,tag:0"`
}

// ParseContentInfo parses DER-encoded ASN.1 data and returns ContentInfo.
func ParseContentInfo(der []byte) (ci ContentInfo, err error) {

	if err != nil {
		return
	}

	var rest []byte
	if rest, err = asn.Unmarshal(der, &ci); err != nil {
		return
	}
	if len(rest) > 0 {
		fmt.Println(ErrTrailingData)
		//err = ErrTrailingData
	}

	return
}

// DER returns the DER-encoded ASN.1 data.
func (ci ContentInfo) DER() ([]byte, error) {
	return asn.Marshal(ci)
}

// Base64 encodes the DER-encoded ASN.1 data in base64 for use in S/MIME.
func (ci ContentInfo) Base64() ([]byte, error) {
	der, err := ci.DER()
	if err != nil {
		return nil, err
	}
	return b64.EncodeBase64(der)
}
