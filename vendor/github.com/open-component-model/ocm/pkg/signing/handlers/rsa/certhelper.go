// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package rsa

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	"github.com/open-component-model/ocm/pkg/signing/signutils"
	"github.com/open-component-model/ocm/pkg/utils"
)

func CreateRootCertificate(sub *pkix.Name, validity time.Duration) (*x509.Certificate, *PrivateKey, error) {
	capriv, _, err := Handler{}.CreateKeyPair()
	if err != nil {
		return nil, nil, err
	}

	spec := &signutils.Specification{
		Subject:      *sub,
		Validity:     validity,
		CAPrivateKey: capriv,
		IsCA:         true,
		Usages:       []interface{}{x509.ExtKeyUsageCodeSigning, x509.KeyUsageDigitalSignature},
	}

	ca, _, err := signutils.CreateCertificate(spec)
	return ca, capriv.(*PrivateKey), err
}

func CreateSigningCertificate(sub *pkix.Name, intermediate signutils.GenericCertificateChain, roots signutils.GenericCertificatePool, capriv signutils.GenericPrivateKey, validity time.Duration, isCA ...bool) (*x509.Certificate, []byte, *PrivateKey, error) {
	priv, pub, err := Handler{}.CreateKeyPair()
	if err != nil {
		return nil, nil, nil, err
	}
	spec := &signutils.Specification{
		IsCA:         utils.Optional(isCA...),
		Subject:      *sub,
		Validity:     validity,
		RootCAs:      roots,
		CAChain:      intermediate,
		CAPrivateKey: capriv,
		PublicKey:    pub,
		Usages:       []interface{}{x509.ExtKeyUsageCodeSigning, x509.KeyUsageDigitalSignature},
	}
	cert, pemBytes, err := signutils.CreateCertificate(spec)
	if err != nil {
		return nil, nil, nil, err
	}
	return cert, pemBytes, priv.(*PrivateKey), nil
}
