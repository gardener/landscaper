// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package rsa

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"

	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

func GetPublicKey(key interface{}) (*rsa.PublicKey, []string, error) {
	var err error
	if data, ok := key.([]byte); ok {
		key, err = ParseKey(data)
		if err != nil {
			return nil, nil, err
		}
	}
	switch k := key.(type) {
	case *rsa.PublicKey:
		return k, nil, nil
	case *rsa.PrivateKey:
		return &k.PublicKey, nil, nil
	case *x509.Certificate:
		if p, ok := k.PublicKey.(*rsa.PublicKey); ok {
			names := slices.Clone(k.DNSNames)
			if k.Issuer.CommonName != "" {
				names = append(names, k.Issuer.CommonName)
			}
			return p, names, nil
		}
		return nil, nil, fmt.Errorf("unknown key public key %T in certificate", k)
	default:
		return nil, nil, fmt.Errorf("unknown key specification %T", k)
	}
}

func GetPrivateKey(key interface{}) (*rsa.PrivateKey, error) {
	if data, ok := key.([]byte); ok {
		return ParsePrivateKey(data)
	}
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return k, nil
	default:
		return nil, fmt.Errorf("unknown key specification %T", k)
	}
}

func WriteKeyData(key interface{}, w io.Writer) error {
	if data, ok := key.([]byte); ok {
		_, err := w.Write(data)
		return err
	}
	block, err := PemBlockForKey(key)
	if err != nil {
		return err
	}
	return pem.Encode(w, block)
}

func KeyData(key interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	block, err := PemBlockForKey(key)
	if err != nil {
		return nil, err
	}
	err = pem.Encode(buf, block)
	return buf.Bytes(), err
}

func PemBlockForKey(priv interface{}, gen ...bool) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PublicKey:
		if utils.Optional(gen...) {
			bytes, err := x509.MarshalPKIXPublicKey(k)
			if err != nil {
				panic(err)
			}
			return &pem.Block{Type: "PUBLIC KEY", Bytes: bytes}, nil
		}
		return &pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(k)}, nil
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	default:
		return nil, errors.ErrInvalid("key")
	}
}

func ParseKey(data []byte) (interface{}, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid key format (expected pem block)")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "CERTIFICATE":
		return x509.ParseCertificate(block.Bytes)
	}
	return ParsePublicKey(data)
}

func ParsePublicKey(data []byte) (interface{}, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid public key format (expected pem block)")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DER encoded public key: %w", err)
		}
	}
	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		return nil, fmt.Errorf("unknown type of public key")
	}
}

func ParsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid private key format (expected pem block)")
	}
	x509Encoded := block.Bytes
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(x509Encoded)
	default:
		untypedPrivateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed parsing key %w", err)
		}
		key, ok := untypedPrivateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("parsed key is not of type *rsa.GetPrivateKey: %T", untypedPrivateKey)
		}
		return key, nil
	}
}
