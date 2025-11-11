// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package signatures

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	cdv2signatures "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/signatures"
)

const (
	// http header
	AcceptHeader             = "Accept"
	HashAlgorithmHeader      = "X-Hash-Algorithm"
	SignatureAlgorithmHeader = "X-Signature-Algorithm"
)

type SigningServerSigner struct {
	ServerURL   string
	ClientCert  *tls.Certificate
	RootCACerts []byte
}

func NewSigningServerSigner(serverURL, clientCertPath, privateKeyPath, rootCACertsPath string) (*SigningServerSigner, error) {
	signer := SigningServerSigner{
		ServerURL: serverURL,
	}

	if clientCertPath != "" {
		clientCert, err := tls.LoadX509KeyPair(clientCertPath, privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load client certificate: %w", err)
		}
		signer.ClientCert = &clientCert
	}

	if rootCACertsPath != "" {
		rootCACerts, err := ioutil.ReadFile(rootCACertsPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read root ca certificates file: %w", err)
		}
		signer.RootCACerts = rootCACerts
	}

	return &signer, nil
}

func (signer *SigningServerSigner) Sign(componentDescriptor cdv2.ComponentDescriptor, digest cdv2.DigestSpec) (*cdv2.SignatureSpec, error) {
	decodedHash, err := hex.DecodeString(digest.Value)
	if err != nil {
		return nil, fmt.Errorf("unable to hex decode hash: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/sign/rsassa-pkcs1-v1_5?hashAlgorithm=%s", signer.ServerURL, digest.HashAlgorithm), bytes.NewBuffer(decodedHash))
	if err != nil {
		return nil, fmt.Errorf("unable to build http request: %w", err)
	}
	req.Header.Add(AcceptHeader, cdv2.MediaTypePEM)

	var certPool *x509.CertPool
	if len(signer.RootCACerts) > 0 {
		certPool = x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM(signer.RootCACerts); !ok {
			return nil, fmt.Errorf("unable to append root ca certificates to cert pool")
		}
	}

	var clientCerts []tls.Certificate
	if signer.ClientCert != nil {
		clientCerts = append(clientCerts, *signer.ClientCert)
	}

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      certPool,
				Certificates: clientCerts,
			},
		},
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send http request: %w", err)
	}
	defer res.Body.Close()

	responseBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request returned with response code %d: %s", res.StatusCode, string(responseBodyBytes))
	}

	signaturePemBlocks, err := cdv2signatures.GetSignaturePEMBlocks(responseBodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed getting signature pem block from response: %w", err)
	}

	if len(signaturePemBlocks) != 1 {
		return nil, fmt.Errorf("expected 1 signature pem block, found %d", len(signaturePemBlocks))
	}
	signatureBlock := signaturePemBlocks[0]

	signature := signatureBlock.Bytes
	if len(signature) == 0 {
		return nil, errors.New("invalid response: signature block doesn't contain signature")
	}

	algorithm := signatureBlock.Headers[cdv2.SignatureAlgorithmHeader]
	if algorithm == "" {
		return nil, fmt.Errorf("invalid response: %s header is empty", cdv2.SignatureAlgorithmHeader)
	}

	encodedSignature := pem.EncodeToMemory(signatureBlock)

	return &cdv2.SignatureSpec{
		Algorithm: algorithm,
		Value:     string(encodedSignature),
		MediaType: cdv2.MediaTypePEM,
	}, nil
}
