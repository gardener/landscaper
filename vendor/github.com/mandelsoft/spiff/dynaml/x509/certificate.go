package x509

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/mandelsoft/spiff/yaml"
	"math/big"
	"net"
	"time"

	. "github.com/mandelsoft/spiff/dynaml"
)

const F_Cert = "x509cert"

func init() {
	RegisterFunction(F_Cert, func_x509cert)
}

//  one map argument with fields
//   usage:        []string
//   organization: []string
//   country: 	   []string (optional)
//   commonName:   string   (optional)
//   validFrom :   string/date   (optional)
//   validity :    int    (hours, optional)
//   isCA:         boolean  (optional)
//   hosts:        []string (optional)
//   privateKey:   string
//   publicKey:    string
//
//   caCert:       string   (optional)
//   caPrivateKey: string   (optional)
//

func func_x509cert(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("invalid argument count for %s(<map>)", F_Cert)
	}
	fields, ok := arguments[0].(map[string]yaml.Node)
	if !ok {
		return info.Error("argument for %s must be a map (found %s)", F_Cert, ExpressionType(arguments[0]))
	}

	isCA, err := getDefaultedBoolField(fields, "isCA", false)
	if err != nil {
		return info.Error(err)
	}

	orgs, err := getStringListField(fields, "organization")
	if err != nil {
		return info.Error(err)
	}

	cn, err := getDefaultedStringField(fields, "commonName", "")
	if err != nil {
		return info.Error(err)
	}

	countries, err := getDefaultedStringListField(fields, "country", nil)
	if err != nil {
		return info.Error(err)
	}

	usages, err := getStringListField(fields, "usage")
	if err != nil {
		return info.Error(err)
	}

	validity, err := getDefaultedIntField(fields, "validity", 24*365)
	if err != nil {
		return info.Error(err)
	}

	hosts, err := getDefaultedStringListField(fields, "hosts", nil)
	if err != nil {
		return info.Error(err)
	}

	privKey, err := getDefaultedStringField(fields, "privateKey", "")
	if err != nil {
		return info.Error(err)
	}
	var priv interface{}
	if privKey != "" {
		priv, err = ParsePrivateKey(privKey)
		if err != nil {
			return info.Error(err)
		}
	}

	pubKey, err := getDefaultedStringField(fields, "publicKey", "")
	if err != nil {
		return info.Error(err)
	}
	var pub interface{}
	if pubKey != "" {
		pub, err = ParsePublicKey(pubKey)
		if err != nil {
			return info.Error(err)
		}
	}

	if pub == nil {
		if priv == nil {
			return info.Error("one of 'publicKey' or 'privateKey' must be given")
		}
		pub = publicKey(priv)
	}

	caCert, err := getDefaultedStringField(fields, "caCert", "")
	if err != nil {
		return info.Error(err)
	}
	var ca *x509.Certificate
	if caCert != "" {
		ca, err = ParseCertificate(caCert)
		if err != nil {
			return info.Error("invalid ca certificate: %s", err)
		}
	}

	caPrivateKey, err := getDefaultedStringField(fields, "caPrivateKey", "")
	if err != nil {
		return info.Error(err)
	}
	var capriv = priv
	if caPrivateKey != "" {
		if ca != nil {
			capriv, err = ParsePrivateKey(caPrivateKey)
			if err != nil {
				return info.Error(err)
			}
		}
	} else {
		if ca != nil {
			return info.Error("private key for ca required")
		}
	}
	if capriv == nil {
		return info.Error("private key for self-signed certificate required")
	}

	var notBefore time.Time
	validFrom, err := getDefaultedStringField(fields, "validFrom", "")
	if err != nil {
		return info.Error(err)
	}
	if validFrom == "" {
		notBefore = time.Now()
	} else {
		notBefore, err = time.Parse("Jan 2 15:04:05 2006", validFrom)
		if err != nil {
			return info.Error("invalid validFrom fields: %s", err)
		}
	}

	notAfter := notBefore.Add(time.Duration(validity) * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return info.Error("failed to generate serial number: %s", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: orgs,
			CommonName:   cn,
			Country:      countries,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              0,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
	}

	if ca == nil {
		ca = template
	}

	for _, u := range usages {
		k := ParseKeyUsage(u)
		if k == nil {
			return info.Error("invalid usage key %q", u)
		}
		k.AddTo(template)
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isCA || (template.KeyUsage&x509.KeyUsageCertSign) != 0 {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, ca, pub, capriv)
	if err != nil {
		return info.Error("Failed to create certificate: %s", err)
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	if err := pem.Encode(writer, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return info.Error("failed to write certificate pem block: %s", err)
	}
	writer.Flush()
	return b.String(), info, true
}
