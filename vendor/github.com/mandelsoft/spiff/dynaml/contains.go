package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
	"strconv"
	"strings"
)

func func_contains(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 2 {
		return info.Error("function contains takes exactly two arguments")
	}

	switch val := arguments[0].(type) {
	case []yaml.Node:
		if arguments[1] == nil {
			return false, info, true
		}

		elem := arguments[1]

		for _, v := range val {
			r, _, _ := compareEquals(v.Value(), elem)
			if r {
				return true, info, true
			}
		}
	case string:
		switch elem := arguments[1].(type) {
		case string:
			return strings.Contains(val, elem), info, true
		case int64:
			return strings.Contains(val, strconv.FormatInt(elem, 10)), info, true
		case bool:
			return strings.Contains(val, strconv.FormatBool(elem)), info, true
		default:
			return info.Error("invalid type for check string")
		}
	default:
		return info.Error("list or string expected for argument one of function contains")
	}
	return false, info, true
}
