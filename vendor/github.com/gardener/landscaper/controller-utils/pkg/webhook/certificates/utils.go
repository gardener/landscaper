// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package certificates

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// EncodePrivateKey takes a RSA private key object, encodes it to the PEM format, and returns it as
// a byte slice.
func EncodePrivateKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// DecodePrivateKey takes a byte slice, decodes it from the PEM format, converts it to an rsa.PrivateKey
// object, and returns it. In case an error occurs, it returns the error.
func DecodePrivateKey(bytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("could not decode the PEM-encoded RSA private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// EncodePrivateKeyInPKCS8 takes a RSA private key object, encodes it to the PKCS8 format, and returns it as
// a byte slice.
func EncodePrivateKeyInPKCS8(key *rsa.PrivateKey) ([]byte, error) {
	bytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bytes,
	}), nil
}

// DecodeRSAPrivateKeyFromPKCS8 takes a byte slice, decodes it from the PKCS8 format, tries to convert it
// to an rsa.PrivateKey object, and returns it. In case an error occurs, it returns the error.
func DecodeRSAPrivateKeyFromPKCS8(bytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("could not decode the PEM-encoded RSA private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("the decoded key is not an RSA private key")
	}
	return rsaKey, nil
}

// EncodeCertificate takes a certificate as a byte slice, encodes it to the PEM format, and returns
// it as byte slice.
func EncodeCertificate(certificate []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificate,
	})
}

// DecodeCertificate takes a byte slice, decodes it from the PEM format, converts it to an x509.Certificate
// object, and returns it. In case an error occurs, it returns the error.
func DecodeCertificate(bytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("could not decode the PEM-encoded certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}
