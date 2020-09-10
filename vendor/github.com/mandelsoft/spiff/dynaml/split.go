package dynaml

import (
	"regexp"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

func func_split(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 2 || len(arguments) > 3 {
		return info.Error("split takes 2 or 3 arguments")
	}

	str, ok := arguments[1].(string)
	if !ok {
		return info.Error("second argument for split must be a string")
	}

	n := -1
	if len(arguments) > 2 {
		m, ok := arguments[2].(int64)
		if !ok {
			return info.Error("third argument for split must be an integer")
		}
		n = int(m)
	}

	var result []yaml.Node
	sep, ok := arguments[0].(string)
	if ok {
		array := strings.SplitN(str, sep, n)
		result = make([]yaml.Node, len(array))
		for i, e := range array {
			result[i] = NewNode(e, binding)
		}
	} else {
		max, ok := arguments[0].(int64)
		if !ok {
			return info.Error("first argument for split must be a string")
		}
		if max <= 0 {
			return info.Error("max line length argument must be positive")
		}
		result = []yaml.Node{}
		cur := ""
		l := int64(0)
		for _, c := range str {
			if l >= max && (n < 0 || n > 1) {
				result = append(result, NewNode(cur, binding))
				cur = ""
				l = 0
				n--
			}
			cur += string(c)
			l++
		}
		if l > 0 {
			result = append(result, NewNode(cur, binding))
		}
	}

	return result, info, true
}

func func_splitMatch(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 2 || len(arguments) > 3 {
		return info.Error("split_match takes 2 or 3 arguments")
	}

	sep, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for split_match must be a string")
	}
	str, ok := arguments[1].(string)
	if !ok {
		return info.Error("second argument for split_match must be a string")
	}

	n := -1
	if len(arguments) > 2 {
		m, ok := arguments[2].(int64)
		if !ok {
			return info.Error("third argument for split must be an integer")
		}
		n = int(m)
	}

	exp, err := regexp.Compile(sep)
	if err != nil {
		return info.Error("split_match: %s", err)
	}
	array := exp.Split(str, n)

	result := make([]yaml.Node, len(array))
	for i, e := range array {
		result[i] = NewNode(e, binding)
	}
	return result, info, true
}
