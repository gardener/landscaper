// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"runtime"
	"time"

	parse "github.com/mandelsoft/spiff/dynaml/x509"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

func GetCertificate(data interface{}) (*x509.Certificate, error) {
	switch k := data.(type) {
	case []byte:
		return ParseCertificate(k)
	case *x509.Certificate:
		return k, nil
	default:
		return nil, fmt.Errorf("unknown type")
	}
}

func ParsePublicKey(data []byte) (interface{}, error) {
	return parse.ParsePublicKey(string(data))
}

func ParsePrivateKey(data []byte) (interface{}, error) {
	return parse.ParsePrivateKey(string(data))
}

func ParseCertificate(data []byte) (*x509.Certificate, error) {
	return parse.ParseCertificate(string(data))
}

func IntermediatePool(pemfile string, fss ...vfs.FileSystem) (*x509.CertPool, error) {
	if pemfile == "" {
		return nil, nil
	}
	fs := accessio.FileSystem(fss...)
	pemdata, err := utils.ReadFile(pemfile, fs)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read cert pem file %q", pemfile)
	}
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(pemdata)
	if !ok {
		return nil, errors.Newf("cannot add cert pem file to cert pool")
	}
	return pool, err
}

func BaseRootPool() (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	sys, err := x509.SystemCertPool()
	if err != nil {
		if runtime.GOOS != "windows" {
			return nil, errors.Wrapf(err, "cannot get system cert pool")
		}
	} else {
		pool = sys
	}
	return pool, nil
}

func RootPool(pemfile string, useOS bool, fss ...vfs.FileSystem) (*x509.CertPool, error) {
	fs := accessio.FileSystem(fss...)
	pemdata, err := utils.ReadFile(pemfile, fs)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read cert pem file %q", pemfile)
	}
	pool := x509.NewCertPool()
	if useOS {
		sys, err := x509.SystemCertPool()
		if err != nil {
			if runtime.GOOS != "windows" {
				return nil, errors.Wrapf(err, "cannot get system cert pool")
			}
		} else {
			pool = sys
		}
	}
	/*
		    cert.RawSubject = subjectSeq
			subjectRDNs, err := parseName(subjectSeq)
			if err != nil {
				return nil, err
			}
			cert.Subject.FillFromRDNSequence(subjectRDNs)
	*/
	ok := pool.AppendCertsFromPEM(pemdata)
	if !ok {
		return nil, errors.Newf("cannot add cert pem file to cert pool")
	}
	return pool, err
}

func VerifyCert(intermediate, root *x509.CertPool, cn string, cert *x509.Certificate) error {
	opts := x509.VerifyOptions{
		DNSName:       cn,
		Intermediates: intermediate,
		Roots:         root,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}
	_, err := cert.Verify(opts)
	if err != nil {
		if cn != "" {
			opts.DNSName = ""
			_, err2 := cert.Verify(opts)
			if err2 == nil && cert.Subject.CommonName == cn {
				return nil
			}
		}
		return err
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

func CreateCertificate(subject pkix.Name, validFrom *time.Time, validity time.Duration,
	pub interface{}, ca *x509.Certificate, priv interface{}, isCA bool, names ...string,
) ([]byte, error) {
	var notBefore time.Time

	if validFrom == nil {
		notBefore = time.Now()
	} else {
		notBefore = *validFrom
	}
	if validity == 0 {
		validity = 24 * 365 * time.Hour
	}
	notAfter := notBefore.Add(validity)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate serial number")
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    notBefore,
		NotAfter:     notAfter,

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
	}

	if ca == nil {
		ca = template
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	found := false
	for _, h := range names {
		if h == subject.CommonName {
			found = true
		}
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}
	if !found {
		template.DNSNames = append(template.DNSNames, subject.CommonName)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, ca, pub, priv)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create certificate")
	}
	return derBytes, err
}
