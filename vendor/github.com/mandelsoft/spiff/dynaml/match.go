package dynaml

import (
	"regexp"
	"strconv"

	"github.com/mandelsoft/spiff/yaml"
)

func func_match(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 2 || len(arguments) > 3 {
		return info.Error("match takes two or three arguments")
	}

	pattern, ok := arguments[0].(string)
	if !ok {
		return info.Error("pattern string for argument one of function match required")
	}

	if arguments[1] == nil {
		return false, info, true
	}

	occ := 0
	if len(arguments) == 3 {
		switch v := arguments[2].(type) {
		case int64:
			occ = int(v)
			if occ == 0 {
				return info.Error("repetition count may not be zero")
			}
		case bool:
			if v {
				occ = -1
			} else {
				occ = 1
			}
		default:
			return info.Error("simple value for argument two of function match required")
		}
	}

	elem := ""
	switch v := arguments[1].(type) {
	case string:
		elem = v
	case int64:
		elem = strconv.FormatInt(v, 10)
	case bool:
		elem = strconv.FormatBool(v)
	default:
		return info.Error("simple value for argument two of function match required")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return info.Error("match: %s", err)
	}

	if occ == 0 {
		list := re.FindStringSubmatch(elem)
		return MakeStringList(list, info), info, true
	} else {
		list := re.FindAllStringSubmatch(elem, occ)
		newList := make([]yaml.Node, len(list))
		for i, v := range list {
			newList[i] = NewNode(MakeStringList(v, info), info)
		}
		return newList, info, true
	}
}

func MakeStringList(list []string, info EvaluationInfo) []yaml.Node {
	newList := make([]yaml.Node, len(list))
	for i, v := range list {
		newList[i] = NewNode(v, info)
	}
	return newList
}
