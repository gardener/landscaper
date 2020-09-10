package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
	"strconv"
	"strings"
)

func func_index(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return _index(true, strings.Index, arguments, binding)
}
func func_lastindex(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return _index(false, strings.LastIndex, arguments, binding)
}

func _index(first bool, f func(string, string) int, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	found := int64(-1)

	if len(arguments) != 2 {
		return info.Error("function index takes exactly two arguments")
	}

	switch val := arguments[0].(type) {
	case []yaml.Node:
		if arguments[1] == nil {
			return -1, info, true
		}

		elem := arguments[1]

		for i, v := range val {
			r, _, _ := compareEquals(v.Value(), elem)
			if r {
				found = int64(i)
				if first {
					break
				}
			}
		}
	case string:
		switch elem := arguments[1].(type) {
		case string:
			return int64(f(val, elem)), info, true
		case int64:
			return int64(f(val, strconv.FormatInt(elem, 10))), info, true
		case bool:
			return int64(f(val, strconv.FormatBool(elem))), info, true
		default:
			return info.Error("invalid type for check string")
		}
	default:
		return info.Error("list or string expected for argument one of function index")
	}
	return int64(found), info, true
}
