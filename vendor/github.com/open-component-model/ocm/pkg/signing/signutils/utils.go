package signutils

import (
	"bytes"
	"crypto"
	"crypto/dsa" //nolint: staticcheck // yes
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/modern-go/reflect2"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
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

func PemBlockForPrivateKey(priv interface{}) *pem.Block {
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

func PemBlockForPublicKey(priv interface{}, gen ...bool) *pem.Block {
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

func ParsePublicKey(data []byte) (interface{}, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid public key format (expected pem block)")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse DER encoded public key")
			}
			pub = cert.PublicKey
		} else {
			return pub, nil
		}
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

func ParsePrivateKey(data []byte) (interface{}, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid private key format (expected pem block)")
	}
	return privateKey(block)
}

func ParseCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block != nil {
		if block.Type != CertificatePEMBlockType {
			return nil, fmt.Errorf("unexpected pem block type for certificate: %q", block.Type)
		}
		return x509.ParseCertificate(block.Bytes)
	}
	return nil, fmt.Errorf("invalid certificate format (expected %s pem block)", CertificatePEMBlockType)
}

func ParseCertificateChain(data []byte, filter bool) ([]*x509.Certificate, error) {
	var chain []*x509.Certificate
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != CertificatePEMBlockType {
			if !filter {
				return nil, fmt.Errorf("unexpected pem block type for certificate: %q", block.Type)
			}
		} else {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			chain = append(chain, cert)
		}
		data = rest
	}

	if len(chain) == 0 {
		if !filter {
			return nil, fmt.Errorf("invalid certificate format (expected CERTIFICATE pem block)")
		}
	}
	return chain, nil
}

type PublicKeySource interface {
	Public() crypto.PublicKey
}

func GetPrivateKey(key GenericPrivateKey) (interface{}, error) {
	switch k := key.(type) {
	case []byte:
		return ParsePrivateKey(k)
	case string:
		return ParsePrivateKey([]byte(k))
	case *rsa.PrivateKey:
		return k, nil
	case *ecdsa.PrivateKey:
		return k, nil
	default:
		return nil, fmt.Errorf("unknown private key specification %T", k)
	}
}

func GetPublicKey(key GenericPublicKey) (interface{}, error) {
	switch k := key.(type) {
	case []byte:
		return ParsePublicKey(k)
	case string:
		return ParsePublicKey([]byte(k))
	case *rsa.PublicKey:
		return k, nil
	case *dsa.PublicKey:
		return k, nil
	case *ecdsa.PublicKey:
		return k, nil
	case *x509.Certificate:
		return k.PublicKey, nil
	case PublicKeySource:
		return k.Public(), nil
	default:
		return nil, fmt.Errorf("unknown public key specification %T", k)
	}
}

func GetCertificateChain(in GenericCertificateChain, filter bool) ([]*x509.Certificate, error) {
	// unfortunately it is not possible to get certificates from a x509.CertPool

	switch k := in.(type) {
	case []byte:
		return ParseCertificateChain(k, filter)
	case string:
		return ParseCertificateChain([]byte(k), filter)
	case *x509.Certificate:
		return []*x509.Certificate{k}, nil
	case []*x509.Certificate:
		return k, nil
	default:
		return nil, fmt.Errorf("unknown certificate chain specification %T", k)
	}
}

////////////////////////////////////////////////////////////////////////////////

func SystemCertPool() (*x509.CertPool, error) {
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

func RootPoolFromFile(pemfile string, useOS bool, fss ...vfs.FileSystem) (*x509.CertPool, error) {
	fs := utils.FileSystem(fss...)
	pemdata, err := utils.ReadFile(pemfile, fs)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read cert pem file %q", pemfile)
	}
	pool := x509.NewCertPool()
	if useOS {
		sys, err := SystemCertPool()
		if err != nil {
			return nil, err
		}
		pool = sys
	}
	ok := pool.AppendCertsFromPEM(pemdata)
	if !ok {
		return nil, errors.Newf("cannot add cert pem file to cert pool")
	}
	return pool, err
}

func GetCertPool(in GenericCertificatePool, filter bool) (*x509.CertPool, error) {
	var certs []*x509.Certificate
	var err error

	if reflect2.IsNil(in) {
		return nil, nil
	}
	switch k := in.(type) {
	case []byte:
		certs, err = ParseCertificateChain(k, filter)
	case string:
		certs, err = ParseCertificateChain([]byte(k), filter)
	case *x509.Certificate:
		certs = []*x509.Certificate{k}
	case []*x509.Certificate:
		certs = k
	case *x509.CertPool:
		return k, nil
	default:
		err = fmt.Errorf("unknown certificate pool specification %T", k)
	}

	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	for _, c := range certs {
		pool.AddCert(c)
	}
	return pool, nil
}

func GetCertificate(in GenericCertificate, filter bool) (*x509.Certificate, *x509.CertPool, error) {
	var certs []*x509.Certificate
	var err error

	switch k := in.(type) {
	case []byte:
		certs, err = ParseCertificateChain(k, filter)
	case string:
		certs, err = ParseCertificateChain([]byte(k), filter)
	case *x509.Certificate:
		certs = []*x509.Certificate{k}
	case []*x509.Certificate:
		certs = k
	default:
		err = fmt.Errorf("unknown certificate chain specification %T", k)
	}

	if err != nil {
		return nil, nil, err
	}

	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no certificate")
	}

	pool := x509.NewCertPool()
	for _, c := range certs {
		pool.AddCert(c)
	}
	return certs[0], pool, nil
}

func IsSelfSigned(cert *x509.Certificate) bool {
	if cert.AuthorityKeyId == nil {
		return true
	}
	return bytes.Equal(cert.SubjectKeyId, cert.AuthorityKeyId)
}

func GetTime(in interface{}) (time.Time, error) {
	switch t := in.(type) {
	case time.Time:
		return t, nil
	case *time.Time:
		return *t, nil
	case string:
		notBefore, err := time.Parse("Jan 2 15:04:05 2006", t)
		if err != nil {
			return time.Time{}, errors.Wrapf(err, "invalid time specification")
		}
		return notBefore, nil
	default:
		return time.Time{}, fmt.Errorf("invalid time specification type %T", in)
	}
}

////////////////////////////////////////////////////////////////////////////////

type KeyUsage interface {
	String() string
	AddTo(*x509.Certificate)
}

type _keyUsage x509.KeyUsage

func (this _keyUsage) AddTo(cert *x509.Certificate) {
	cert.KeyUsage |= x509.KeyUsage(this)
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

func GetKeyUsage(opt interface{}) KeyUsage {
	switch o := opt.(type) {
	case x509.ExtKeyUsage:
		return _extKeyUsage(o)
	case x509.KeyUsage:
		return _keyUsage(o)
	case string:
		return ParseKeyUsage(o)
	default:
		return nil
	}
}
