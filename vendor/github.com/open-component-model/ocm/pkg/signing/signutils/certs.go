// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signutils

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"time"

	"github.com/modern-go/reflect2"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type Usages []interface{}

// Specification specified the context for the certificate creation.
type Specification struct {
	// RootCAs is used to verify a certificate chain.
	// Self-signed CAs must be added here to be accepted as part
	// of a chain.
	RootCAs GenericCertificatePool

	// IsCA requests a certificate for a CA.
	IsCA bool

	PublicKey GenericPublicKey

	// CAPrivateKey is the private key used for signing.
	// It must be the key for the first certificate in the chain
	// (if given).
	CAPrivateKey GenericPrivateKey
	CAChain      GenericCertificateChain

	// SkipVerify can be set to true to skip the verification
	// of the given certificate chain.
	SkipVerify bool

	Subject   pkix.Name
	Usages    Usages
	Validity  time.Duration
	NotBefore *time.Time

	Hosts []string
}

// CreateCertificate creates a certificate and additionally returns a PEM encoded
// representation.
func CreateCertificate(spec *Specification) (*x509.Certificate, []byte, error) {
	var err error

	var rootCerts *x509.CertPool

	if !reflect2.IsNil(spec.RootCAs) {
		rootCerts, err = GetCertPool(spec.RootCAs, true)
		if err != nil {
			return nil, nil, err
		}
	}

	var pubKey interface{}
	if !reflect2.IsNil(spec.PublicKey) {
		pubKey, err = GetPublicKey(spec.PublicKey)
		if err != nil {
			return nil, nil, err
		}
	}

	var caChain []*x509.Certificate
	if !reflect2.IsNil(spec.CAChain) {
		caChain, err = GetCertificateChain(spec.CAChain, false)
		if err != nil {
			return nil, nil, err
		}
	}

	var caPrivKey interface{}
	if !reflect2.IsNil(spec.CAPrivateKey) {
		caPrivKey, err = GetPrivateKey(spec.CAPrivateKey)
		if err != nil {
			return nil, nil, err
		}
	}
	if reflect2.IsNil(caPrivKey) {
		return nil, nil, fmt.Errorf("private key required for signing")
	}

	if reflect2.IsNil(pubKey) {
		pubKey, err = GetPublicKey(caPrivKey)
		if err != nil {
			return nil, nil, err
		}
	}

	var notBefore time.Time
	if spec.NotBefore == nil {
		notBefore = time.Now()
	} else {
		notBefore, err = GetTime(spec.NotBefore)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(caChain) > 0 {
		key, ok := caPrivKey.(crypto.Signer)
		if !ok {
			return nil, nil, errors.Newf("x509: certificate private key does not implement crypto.Signer")
		}
		if !reflect.DeepEqual(key.Public(), caChain[0].PublicKey) {
			return nil, nil, errors.Newf("private key does not match ca certificate")
		}
		if !spec.SkipVerify {
			if rootCerts == nil {
				rootCerts, err = x509.SystemCertPool()
				if err != nil {
					return nil, nil, err
				}
			}

			intermediates, err := GetCertPool(caChain, false)
			if err != nil {
				return nil, nil, err
			}

			opts := x509.VerifyOptions{
				Intermediates:             intermediates,
				Roots:                     rootCerts,
				KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny}, // x509.ExtKeyUsageCodeSigning ???
				MaxConstraintComparisions: 0,
			}

			_, err = caChain[0].Verify(opts)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	validity := spec.Validity
	if validity == 0 {
		validity = time.Hour*24 + 365
	}
	notAfter := notBefore.Add(validity)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate serial number")
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      spec.Subject,
		NotBefore:    notBefore,
		NotAfter:     notAfter,

		KeyUsage:              0,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
	}

	var ca *x509.Certificate
	if len(caChain) == 0 {
		ca = template // se0fl signed certificate
	} else {
		ca = caChain[0]
	}

	for _, u := range spec.Usages {
		k := GetKeyUsage(u)
		if k == nil {
			return nil, nil, fmt.Errorf("invalid usage key %q", u)
		}
		k.AddTo(template)
	}

	for _, h := range spec.Hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if spec.IsCA || (template.KeyUsage&x509.KeyUsageCertSign) != 0 {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, ca, pubKey, caPrivKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create certificate")
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		panic("failed to parse generated certificate:" + err.Error())
	}

	pemBytes := CertificateBytesToPem(derBytes)
	for _, c := range caChain {
		pemBytes = append(pemBytes, CertificateToPem(c)...)
	}
	return cert, pemBytes, nil
}

func VerifyCertificate(cert *x509.Certificate, intermediates GenericCertificateChain, rootCerts GenericCertificatePool, name *pkix.Name, ts ...*time.Time) error {
	rootPool, err := GetCertPool(rootCerts, false)
	if err != nil {
		return err
	}
	interPool, err := GetCertPool(intermediates, false)
	if err != nil {
		return err
	}
	timestamp := cert.NotBefore
	if ts := utils.Optional(ts...); ts != nil && !ts.IsZero() {
		timestamp = *ts
	}
	opts := x509.VerifyOptions{
		Intermediates:             interPool,
		Roots:                     rootPool,
		CurrentTime:               timestamp,
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		MaxConstraintComparisions: 0,
	}

	_, err = cert.Verify(opts)
	if err != nil {
		return err
	}
	if name != nil {
		return errors.Wrapf(MatchDN(cert.Subject, *name), "issuer mismatch in public key certificate")
	}
	return nil
}

func CertificateChainToPem(certs []*x509.Certificate) []byte {
	var data []byte
	for _, c := range certs {
		data = append(data, CertificateToPem(c)...)
	}
	return data
}

func CertificateToPem(c *x509.Certificate) []byte {
	return CertificateBytesToPem(c.Raw)
}

func CertificateBytesToPem(derBytes []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: CertificatePEMBlockType, Bytes: derBytes})
}
