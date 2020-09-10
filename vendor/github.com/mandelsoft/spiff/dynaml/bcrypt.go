package dynaml

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func func_bcrypt(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	cost := 10

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("bcrypt takes one or two arguments")
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for bcrypt must be a string")
	}

	if len(arguments) > 1 {
		c, ok := arguments[1].(int64)
		if !ok {
			return info.Error("second argument for bcrypt must be an integer")
		}
		cost = int(c)
	}
	result, err := bcrypt.GenerateFromPassword([]byte(str), cost)
	if err != nil {
		return info.Error("bcrypt error: %s", err)
	}

	return fmt.Sprintf("%s", result), info, true
}

func func_bcrypt_check(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 2 {
		return info.Error("bcrypt_check takes two arguments")
	}

	passwd, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for bcrypt_check must be a string")
	}

	hash, ok := arguments[1].(string)
	if !ok {
		return info.Error("second argument for bcrypt_check must be a string")
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(passwd))
	return err == nil, info, true
}
