package dynaml

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"golang.org/x/crypto/md4"
)

func func_md5(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("md5 takes exactly one argument")
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for md5 must be a string")
	}

	result := md5.Sum([]byte(str))
	return fmt.Sprintf("%x", result), info, true
}

func func_hash(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("hash takes one or two arguments")
	}

	mode := "sha256"

	if len(arguments) == 2 {
		str, ok := arguments[1].(string)
		if !ok {
			return info.Error("second argument for hash must be a string")
		}
		mode = str
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for hash must be a string")
	}

	var result []byte
	switch mode {
	case "md4":
		result = md4.New().Sum([]byte(str))
	case "md5":
		r := md5.Sum([]byte(str))
		result = r[:]
	case "sha1":
		r := sha1.Sum([]byte(str))
		result = r[:]
	case "sha224":
		r := sha256.Sum224([]byte(str))
		result = r[:]
	case "sha256":
		r := sha256.Sum256([]byte(str))
		result = r[:]
	case "sha384":
		r := sha512.Sum384([]byte(str))
		result = r[:]
	case "sha512":
		r := sha512.Sum512([]byte(str))
		result = r[:]
	case "sha512/224":
		r := sha512.Sum512_224([]byte(str))
		result = r[:]
	case "sha512/256":
		r := sha512.Sum512_256([]byte(str))
		result = r[:]
	default:
		return info.Error("invalid hash type '%s'", mode)
	}
	return fmt.Sprintf("%x", result), info, true
}
