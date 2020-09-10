package x509

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"strconv"

	. "github.com/mandelsoft/spiff/dynaml"
)

const F_GenKey = "x509genkey"

func init() {
	RegisterFunction(F_GenKey, func_x509genkey)
}

// one optional argument
//  - either rsa bit size (int)
//  - or ecdsaCurve (string)

func func_x509genkey(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error
	var ok bool
	info := DefaultInfo()
	ecdsaCurve := ""
	rsaBits := int64(2048)

	if len(arguments) > 1 {
		return info.Error("invalid argument count for %s([<bitsize>|<ecdsaCurve>])", F_GenKey)
	}

	if len(arguments) > 0 {
		rsaBits, ok = arguments[0].(int64)
		if !ok {
			str, ok := arguments[0].(string)
			if !ok {
				return info.Error("argument for %s must be a string or integer", F_GenKey)
			}
			rsaBits, err = strconv.ParseInt(str, 10, 32)
			if err != nil {
				ecdsaCurve = str
			}
		}
	}

	var priv interface{}
	switch ecdsaCurve {
	case "":
		priv, err = rsa.GenerateKey(rand.Reader, int(rsaBits))
	case "P224":
		priv, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case "P256":
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "P384":
		priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "P521":
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		return info.Error("Unrecognized elliptic curve: %q", ecdsaCurve)
	}
	if err != nil {
		return info.Error("failed to generate private key: %s", err)
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	if err := pem.Encode(writer, pemBlockForKey(priv)); err != nil {
		return info.Error("failed to write key pem block: %s", err)
	}
	writer.Flush()
	return b.String(), info, true
}
