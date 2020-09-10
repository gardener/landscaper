package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/dynaml/crypt"
	"strings"
)

func func_md5crypt(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("md5crypt takes one argument")
	}

	passwd, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for md5crypt must be a string")
	}

	result := crypt.MD5Crypt([]byte(passwd), crypt.GenerateSALT(8), []byte(crypt.MD5_MAGIC))

	return fmt.Sprintf("%s", result), info, true
}

func func_md5crypt_check(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 2 {
		return info.Error("md5crypt_check takes two arguments")
	}

	passwd, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for md5crypt_check must be a string")
	}

	hash, ok := arguments[1].(string)
	if !ok {
		return info.Error("second argument for md5crypt_check must be a string")
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 4 {
		return info.Error("invalid md5crypt hash: must contain three '$' characters")
	}
	check := crypt.MD5Crypt([]byte(passwd), []byte(parts[2]), []byte("$"+parts[1]+"$"))
	return string(check) == hash, info, true
}
