package timestamp

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"time"

	asn "github.com/InfiniteLoopSpace/go_S-MIME/asn1"
	cms "github.com/InfiniteLoopSpace/go_S-MIME/cms/protocol"
	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

// TSTInfo ::= SEQUENCE  {
//    version                      INTEGER  { v1(1) },
//    policy                       TSAPolicyId,
//    messageImprint               MessageImprint,
//      -- MUST have the same value as the similar field in
//      -- TimeStampReq
//    serialNumber                 INTEGER,
//     -- Time-Stamping users MUST be ready to accommodate integers
//     -- up to 160 bits.
//    genTime                      GeneralizedTime,
//    accuracy                     Accuracy                 OPTIONAL,
//    ordering                     BOOLEAN             DEFAULT FALSE,
//    nonce                        INTEGER                  OPTIONAL,
//      -- MUST be present if the similar field was present
//      -- in TimeStampReq.  In that case it MUST have the same value.
//    tsa                          [0] GeneralName          OPTIONAL,
//    extensions                   [1] IMPLICIT Extensions  OPTIONAL   }
type TSTInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint MessageImprint
	SerialNumber   *big.Int
	GenTime        time.Time        `asn1:"generalized"`
	Accuracy       Accuracy         `asn1:"optional"`
	Ordering       bool             `asn1:"optional,default:false"`
	Nonce          *big.Int         `asn1:"optional"`
	TSA            asn1.RawValue    `asn1:"tag:0,optional"`
	Extensions     []pkix.Extension `asn1:"tag:1,optional"`
}

// ParseInfo parses an Info out of a CMS EncapsulatedContentInfo.
func ParseInfo(enci cms.EncapsulatedContentInfo) (TSTInfo, error) {
	i := TSTInfo{}
	if !enci.EContentType.Equal(oid.TSTInfo) {
		return i, cms.ErrWrongType
	}

	if rest, err := asn.Unmarshal(enci.EContent, &i); err != nil {
		return i, err
	} else if len(rest) > 0 {
		return i, cms.ErrTrailingData
	}

	return i, nil
}

// Before checks if the latest time the signature could have been generated at
// is before the specified time. For example, you might check that a signature
// was made *before* a certificate's not-after date.
func (i *TSTInfo) Before(t time.Time) bool {
	return i.genTimeMax().Before(t)
}

// After checks if the earlier time the signature could have been generated at
// is before the specified time. For example, you might check that a signature
// was made *after* a certificate's not-before date.
func (i *TSTInfo) After(t time.Time) bool {
	return i.genTimeMin().After(t)
}

// genTimeMax is the latest time at which the token could have been generated
// based on the included GenTime and Accuracy attributes.
func (i *TSTInfo) genTimeMax() time.Time {
	return i.GenTime.Add(i.Accuracy.Duration())
}

// genTimeMin is the earliest time at which the token could have been generated
// based on the included GenTime and Accuracy attributes.
func (i *TSTInfo) genTimeMin() time.Time {
	return i.GenTime.Add(-i.Accuracy.Duration())
}

// Accuracy of the timestamp
type Accuracy struct {
	Seconds int `asn1:"optional"`
	Millis  int `asn1:"tag:0,optional"`
	Micros  int `asn1:"tag:1,optional"`
}

// Duration returns this Accuracy as a time.Duration.
func (a Accuracy) Duration() time.Duration {
	return 0 +
		time.Duration(a.Seconds)*time.Second +
		time.Duration(a.Millis)*time.Millisecond +
		time.Duration(a.Micros)*time.Microsecond
}
