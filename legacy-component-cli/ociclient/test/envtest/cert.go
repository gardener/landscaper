// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"net"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Certificate defines a certificate with a CA.
// All certs and keys are PEM encoded.
type Certificate struct {
	CA    []byte
	CAKey []byte
	Cert  []byte
	Key   []byte
}

// GenerateCertificates generates a new ca and signed cert.
// Generated cert is only valid for 127.0.0.1.
// Only use this for testing as insecure keys are used.
func GenerateCertificates() (*Certificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"testing"},
			Country:      []string{"DE"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 1024) // only for testing
	if err != nil {
		return nil, fmt.Errorf("unable to generate private key: %w", err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("unable to create ca certificate: %w", err)
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to PEM encode ca: %w", err)
	}

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to PEM encode ca private key: %w", err)
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"testing"},
			Country:      []string{"DE"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certPrivKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, fmt.Errorf("unable to generate private key: %w", err)
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("unable to create certificate: %w", err)
	}

	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to PEM encode cert: %w", err)
	}

	certPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to PEM encode private key: %w", err)
	}

	return &Certificate{
		CA:    caPEM.Bytes(),
		CAKey: caPrivKeyPEM.Bytes(),
		Cert:  certPEM.Bytes(),
		Key:   certPrivKeyPEM.Bytes(),
	}, nil
}

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	mathrand.Seed(time.Now().Unix())
}

// RandString creates a random string with n characters.
func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[mathrand.Intn(len(chars))]
	}
	return string(b)
}

// CreateHtpasswd creates a htpasswd string for a username and password
func CreateHtpasswd(username, password string) string {
	// docker registry only allows bcrypt htpasswd
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return fmt.Sprintf("%s:%s", username, string(hash))
}
