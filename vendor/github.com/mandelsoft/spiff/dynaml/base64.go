package dynaml

import (
	"encoding/base64"
	"strconv"
	"strings"
)

func func_base64(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error

	info := DefaultInfo()

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("base64 takes one or two argumenta")
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for base64 must be a string")
	}

	max := -1
	if len(arguments) > 1 {
		l, ok := arguments[1].(int64)
		if !ok {
			s, ok := arguments[1].(string)
			if !ok {
				return info.Error("second argument for base64 must be an integer or interger string")
			}
			l, err = strconv.ParseInt(s, 10, 64)
			if err != nil {
				return info.Error("second argument for base64 must be an integer or interger string: %s", err)
			}
		}
		max = int(l)
	}
	result := Base64Encode([]byte(str), max)
	return result, info, true
}

func func_base64_decode(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("base64_decode takes exactly one argument")
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for base64_decode must be a string")
	}

	result, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return info.Error("cannot decode string")
	}
	return string(result), info, true
}

func Base64Encode(data []byte, max int) string {
	str := base64.StdEncoding.EncodeToString(data)
	if max > 0 {
		result := ""
		for len(str) > max {
			result = result + str[:max] + "\n"
			str = str[max:]
		}
		if len(str) > 0 {
			result = result + str
		}
		if strings.HasSuffix(result, "\n") {
			result = result[:len(result)-1]
		}
		return result
	} else {
		return str
	}
}
