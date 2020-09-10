package x509

import (
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strings"
)

func privateKey(block *pem.Block) (interface{}, error) {
	x509Encoded := block.Bytes
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(x509Encoded)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(x509Encoded)
	default:
		return nil, fmt.Errorf("invalid pem block type %q", block.Type)
	}
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PublicKey:
		return k
	case *ecdsa.PublicKey:
		return k

	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case *x509.Certificate:
		return k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		log.Fatal("invalid key")
		return nil
	}
}

func pemBlockForPublicKey(priv interface{}, gen ...bool) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PublicKey:
		if len(gen) > 0 && gen[0] {
			bytes, err := x509.MarshalPKIXPublicKey(k)
			if err != nil {
				panic(err)
			}
			return &pem.Block{Type: "PUBLIC KEY", Bytes: bytes}
		}
		return &pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(k)}
	case *ecdsa.PublicKey:
		b, err := x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return nil
		}
		return &pem.Block{Type: "ECDSA PUBLIC KEY", Bytes: b}
	default:
		return nil
	}
}

func ParsePublicKey(data string) (interface{}, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("invalid public key format (expected pem block)")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DER encoded public key: %s", err)
		}
		return pub, nil
	}
	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	case *dsa.PublicKey:
		return pub, nil
	case *ecdsa.PublicKey:
		return pub, nil
	default:
		return nil, fmt.Errorf("unknown type of public key")
	}
}

func ParsePrivateKey(data string) (interface{}, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("invalid private key format (expected pem block)")
	}
	return privateKey(block)
}

func ParseCertificate(data string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("invalid certificate format (expected pem block)")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("unexpected pem block type for certificate: %q", block.Type)
	}
	return x509.ParseCertificate(block.Bytes)
}

////////////////////////////////////////////////////////////////////////////////

type KeyUsage interface {
	String() string
	AddTo(*x509.Certificate)
}

type _keyUsage x509.KeyUsage

func (this _keyUsage) AddTo(cert *x509.Certificate) {
	cert.KeyUsage = cert.KeyUsage | x509.KeyUsage(this)
}

func (this _keyUsage) String() string {
	switch x509.KeyUsage(this) {
	case x509.KeyUsageDigitalSignature:
		return "Signature"
	case x509.KeyUsageContentCommitment:
		return "ContentCommitment"
	case x509.KeyUsageKeyEncipherment:
		return "KeyEncipherment"
	case x509.KeyUsageDataEncipherment:
		return "DataEncipherment"
	case x509.KeyUsageKeyAgreement:
		return "KeyAgreement"
	case x509.KeyUsageCertSign:
		return "CertSign"
	case x509.KeyUsageCRLSign:
		return "CRLSign"
	case x509.KeyUsageEncipherOnly:
		return "EncipherOnly"
	case x509.KeyUsageDecipherOnly:
		return "DecipherOnly"
	default:
		return "UnknownKeyUsage"
	}
}

var _keyUsages = []x509.KeyUsage{
	x509.KeyUsageDigitalSignature,
	x509.KeyUsageContentCommitment,
	x509.KeyUsageKeyEncipherment,
	x509.KeyUsageDataEncipherment,
	x509.KeyUsageKeyAgreement,
	x509.KeyUsageCertSign,
	x509.KeyUsageCRLSign,
	x509.KeyUsageEncipherOnly,
	x509.KeyUsageDecipherOnly,
}

func KeyUsages(usages x509.KeyUsage) []string {
	result := []string{}
	for _, u := range _keyUsages {
		if usages&u != 0 {
			result = append(result, (_keyUsage(u)).String())
		}
	}
	return result
}

type _extKeyUsage x509.ExtKeyUsage

func (this _extKeyUsage) AddTo(cert *x509.Certificate) {
	for _, k := range cert.ExtKeyUsage {
		if k == x509.ExtKeyUsage(this) {
			return
		}
	}
	cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsage(this))
}

func (this _extKeyUsage) String() string {
	switch x509.ExtKeyUsage(this) {
	case x509.ExtKeyUsageAny:
		return "Any"
	case x509.ExtKeyUsageServerAuth:
		return "ServerAuth"
	case x509.ExtKeyUsageClientAuth:
		return "ClientAuth"
	case x509.ExtKeyUsageCodeSigning:
		return "CodeSigning"
	case x509.ExtKeyUsageEmailProtection:
		return "EmailProtection"
	case x509.ExtKeyUsageIPSECEndSystem:
		return "IPSECEndSystem"
	case x509.ExtKeyUsageIPSECTunnel:
		return "IPSECTunnel"
	case x509.ExtKeyUsageIPSECUser:
		return "IPSECUser"
	case x509.ExtKeyUsageTimeStamping:
		return "TimeStamping"
	case x509.ExtKeyUsageOCSPSigning:
		return "OCSPSigning"
	case x509.ExtKeyUsageMicrosoftServerGatedCrypto:
		return "MicrosoftServerGatedCrypto"
	case x509.ExtKeyUsageNetscapeServerGatedCrypto:
		return "NetscapeServerGatedCrypto"
	case x509.ExtKeyUsageMicrosoftCommercialCodeSigning:
		return "MicrosoftCommercialCodeSigning"
	case x509.ExtKeyUsageMicrosoftKernelCodeSigning:
		return "MicrosoftKernelCodeSigning"
	default:
		return "UnknownExtKeyUsage"
	}
}

func ExtKeyUsages(usages []x509.ExtKeyUsage) []string {
	result := []string{}
	for _, u := range usages {
		result = append(result, (_extKeyUsage(u)).String())
	}
	return result
}

func ParseKeyUsage(name string) KeyUsage {
	switch strings.ToLower(name) {
	case "signature":
		return _keyUsage(x509.KeyUsageDigitalSignature)
	case "commitment":
		return _keyUsage(x509.KeyUsageContentCommitment)
	case "keyencipherment":
		return _keyUsage(x509.KeyUsageKeyEncipherment)
	case "dataencipherment":
		return _keyUsage(x509.KeyUsageDataEncipherment)
	case "keyagreement":
		return _keyUsage(x509.KeyUsageKeyAgreement)
	case "certsign":
		return _keyUsage(x509.KeyUsageCertSign)
	case "crlsign":
		return _keyUsage(x509.KeyUsageCRLSign)
	case "encipheronly":
		return _keyUsage(x509.KeyUsageEncipherOnly)
	case "decipheronly":
		return _keyUsage(x509.KeyUsageDecipherOnly)

	case "any":
		return _extKeyUsage(x509.ExtKeyUsageAny)
	case "serverauth":
		return _extKeyUsage(x509.ExtKeyUsageServerAuth)
	case "clientauth":
		return _extKeyUsage(x509.ExtKeyUsageClientAuth)
	case "codesigning":
		return _extKeyUsage(x509.ExtKeyUsageCodeSigning)
	case "emailprotection":
		return _extKeyUsage(x509.ExtKeyUsageEmailProtection)
	case "ipsecendsystem":
		return _extKeyUsage(x509.ExtKeyUsageIPSECEndSystem)
	case "ipsectunnel":
		return _extKeyUsage(x509.ExtKeyUsageIPSECTunnel)
	case "ipsecuser":
		return _extKeyUsage(x509.ExtKeyUsageIPSECUser)
	case "timestamping":
		return _extKeyUsage(x509.ExtKeyUsageTimeStamping)
	case "ocspsigning":
		return _extKeyUsage(x509.ExtKeyUsageOCSPSigning)
	case "microsoftservergatedcrypto":
		return _extKeyUsage(x509.ExtKeyUsageMicrosoftServerGatedCrypto)
	case "netscapeservergatedcrypto":
		return _extKeyUsage(x509.ExtKeyUsageNetscapeServerGatedCrypto)
	case "microsoftcommercialcodesigning":
		return _extKeyUsage(x509.ExtKeyUsageMicrosoftCommercialCodeSigning)
	case "microsoftkernelcodesigning":
		return _extKeyUsage(x509.ExtKeyUsageMicrosoftKernelCodeSigning)
	}
	return nil
}
