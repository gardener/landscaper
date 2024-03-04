package protocol

import (
	"bytes"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"math/big"
)

// IssuerAndSerialNumber ::= SEQUENCE {
// 	issuer Name,
// 	serialNumber CertificateSerialNumber }
//
// CertificateSerialNumber ::= INTEGER
type IssuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

// NewIssuerAndSerialNumber creates a IssuerAndSerialNumber SID for the given
// cert.
func NewIssuerAndSerialNumber(cert *x509.Certificate) (sid IssuerAndSerialNumber, err error) {
	sid = IssuerAndSerialNumber{
		SerialNumber: new(big.Int).Set(cert.SerialNumber),
	}

	if _, err = asn1.Unmarshal(cert.RawIssuer, &sid.Issuer); err != nil {
		return
	}

	return
}

// RawValue returns the RawValue of the IssuerAndSerialNumber.
func (ias *IssuerAndSerialNumber) RawValue() (rv asn1.RawValue, err error) {
	var der []byte
	if der, err = asn1.Marshal(*ias); err != nil {
		return
	}

	if _, err = asn1.Unmarshal(der, &rv); err != nil {
		return
	}

	return
}

// Equal returns true if ias and ias2 agree.
func (ias *IssuerAndSerialNumber) Equal(ias2 IssuerAndSerialNumber) bool {

	if bytes.Compare(ias.Issuer.Bytes, ias2.Issuer.Bytes) != 0 {
		return false
	}

	if ias.SerialNumber.Cmp(ias2.SerialNumber) != 0 {
		return false
	}

	return true
}

// IASstring retuns the ias of the cert as hex encoded string.
func IASstring(cert *x509.Certificate) (iasString string, err error) {
	ias, err := NewIssuerAndSerialNumber(cert)
	if err != nil {
		return
	}

	rv, err := ias.RawValue()
	if err != nil {
		return
	}

	iasString = fmt.Sprintf("%x", rv.Bytes)
	return
}
