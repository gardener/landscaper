package x509

import (
	"bufio"
	"bytes"
	"encoding/pem"
	"fmt"
	. "github.com/mandelsoft/spiff/dynaml"
	"strings"

	"golang.org/x/crypto/ssh"
)

const F_PublicKey = "x509publickey"

func init() {
	RegisterFunction(F_PublicKey, func_x509publickey)
}

// one argument
//  - private key pem

func func_x509publickey(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error
	info := DefaultInfo()

	rtype := "pem"
	switch len(arguments) {
	case 1:
	case 2:
		str, ok := arguments[1].(string)
		if !ok {
			return info.Error("format argument for %s must be a string", F_PublicKey)
		}
		lower := strings.ToLower(str)
		switch lower {
		case "pem":
		case "ssh", "pkix":
			rtype = lower
		default:
			return info.Error("invalid format for %s: %s", F_PublicKey, str)
		}

	default:
		return info.Error("invalid argument count for %s", F_PublicKey)
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("argument for %s must be a private key in pem format", F_PublicKey)
	}

	key, err := ParsePrivateKey(str)
	if err != nil {
		k, e := ParsePublicKey(str)
		if e != nil {
			return info.Error("argument for %s must be a private key in pem format: %s", F_PublicKey, err)
		}
		key = k
	}

	switch rtype {
	case "pem":
		str, err = PublicKeyPEM(publicKey(key))
	case "pkix":
		str, err = PublicKeyPEM(publicKey(key), true)
	case "ssh":
		var pk ssh.PublicKey
		pk, err = ssh.NewPublicKey(publicKey(key))
		if err == nil {
			str = string(ssh.MarshalAuthorizedKey(pk))
		}
	}

	if err != nil {
		return info.Error("%s", err)
	}
	return str, info, true
}

func PublicKeyPEM(key interface{}, gen ...bool) (string, error) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	if err := pem.Encode(writer, pemBlockForPublicKey(key, gen...)); err != nil {
		return "", fmt.Errorf("failed to write public key pem block: %s", err)
	}
	writer.Flush()
	return b.String(), nil
}
