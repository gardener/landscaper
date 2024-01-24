// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signutils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// MediaTypePEM defines the media type for PEM formatted signature data.
const MediaTypePEM = "application/x-pem-file"

// SignaturePEMBlockType defines the type of a signature pem block.
const SignaturePEMBlockType = "SIGNATURE"

// CertificatePEMBlockType defines the type of a certificate pem block.
const CertificatePEMBlockType = "CERTIFICATE"

// SignaturePEMBlockAlgorithmHeader defines the header in a signature pem block where the signature algorithm is defined.
const SignaturePEMBlockAlgorithmHeader = "Signature Algorithm"

// GetSignatureFromPem returns a signature and certificated contained
// in a PEM block list.
func GetSignatureFromPem(pemData []byte) ([]byte, string, []*x509.Certificate, error) {
	var signature []byte
	var algo string

	if len(pemData) == 0 {
		return nil, "", nil, nil
	}

	data := pemData
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			return nil, "", nil, fmt.Errorf("PEM des not contain signature")
		}
		if block == nil && len(data) > 0 {
			return nil, "", nil, fmt.Errorf("unable to decode pem block %s", string(data))
		}
		if block.Type == SignaturePEMBlockType {
			signature = block.Bytes
			algo = block.Headers[SignaturePEMBlockAlgorithmHeader]
			break
		}
		data = rest
	}

	caChain, err := ParseCertificateChain(pemData, true)
	if err != nil {
		return nil, "", nil, err
	}
	return signature, algo, caChain, nil
}

func SignatureBytesToPem(algo string, data []byte, certs ...*x509.Certificate) []byte {
	block := &pem.Block{Type: SignaturePEMBlockType, Bytes: data}
	if algo != "" {
		block.Headers = map[string]string{
			SignaturePEMBlockAlgorithmHeader: algo,
		}
	}
	return append(pem.EncodeToMemory(block), CertificateChainToPem(certs)...)
}
