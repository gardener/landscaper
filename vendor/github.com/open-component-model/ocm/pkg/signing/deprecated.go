// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"crypto/x509"

	parse "github.com/mandelsoft/spiff/dynaml/x509"

	"github.com/open-component-model/ocm/pkg/signing/signutils"
)

// Deprecated: use signutils.GetCertificate.
func GetCertificate(in interface{}) (*x509.Certificate, error) {
	c, _, err := signutils.GetCertificate(in, false)
	return c, err
}

// Deprecated: use signutils.ParsePublicKey.
func ParsePublicKey(data []byte) (interface{}, error) {
	return parse.ParsePublicKey(string(data))
}

// Deprecated: use signutils.ParsePrivateKey.
func ParsePrivateKey(data []byte) (interface{}, error) {
	return parse.ParsePrivateKey(string(data))
}

// Deprecated: use signutils.SystemCertPool.
func BaseRootPool() (*x509.CertPool, error) {
	return signutils.SystemCertPool()
}
