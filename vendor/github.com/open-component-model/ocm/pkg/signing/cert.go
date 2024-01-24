// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing/signutils"
)

func VerifyCert(intermediate signutils.GenericCertificateChain, root signutils.GenericCertificatePool, name string, cert *x509.Certificate) error {
	return VerifyCertDN(intermediate, root, signutils.CommonName(name), cert)
}

func VerifyCertDN(intermediate signutils.GenericCertificateChain, root signutils.GenericCertificatePool, name *pkix.Name, cert *x509.Certificate) error {
	rootPool, err := signutils.GetCertPool(root, false)
	if err != nil {
		return err
	}
	interPool, err := signutils.GetCertPool(intermediate, false)
	if err != nil {
		return err
	}
	opts := x509.VerifyOptions{
		Intermediates: interPool,
		Roots:         rootPool,
		CurrentTime:   cert.NotBefore,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}
	_, err = cert.Verify(opts)
	if err != nil {
		return err
	}
	if name != nil {
		if err := signutils.MatchDN(cert.Subject, *name); err != nil {
			return err
		}
	}
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		return nil
	}
	for _, k := range cert.ExtKeyUsage {
		if k == x509.ExtKeyUsageCodeSigning {
			return nil
		}
	}
	return errors.ErrNotSupported("codesign", "", "certificate")
}

// Deprecated: use signutils.CreateCertificate.
func CreateCertificate(subject pkix.Name, validFrom *time.Time,
	validity time.Duration, pub interface{},
	ca *x509.Certificate, priv interface{}, isCA bool, names ...string,
) ([]byte, error) {
	spec := &signutils.Specification{
		RootCAs:      ca,
		IsCA:         isCA,
		PublicKey:    pub,
		CAPrivateKey: priv,
		CAChain:      ca,
		Subject:      subject,
		Usages:       signutils.Usages{x509.ExtKeyUsageCodeSigning},
		Validity:     validity,
		NotBefore:    validFrom,
		Hosts:        names,
	}
	_, data, err := signutils.CreateCertificate(spec)
	return data, err
}
