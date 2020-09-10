package dynaml

import (
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

func func_trim(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	ok := true

	if len(arguments) > 2 {
		return info.Error("split takes one or two arguments")
	}

	cutset := " \t"
	if len(arguments) == 2 {
		cutset, ok = arguments[1].(string)
		if !ok {
			return info.Error("second argument of split must be a string")
		}
	}
	var result interface{}
	switch v := arguments[0].(type) {
	case string:
		result = strings.Trim(v, cutset)
	case []yaml.Node:
		list := make([]yaml.Node, len(v))
		for i, e := range v {
			t, ok := e.Value().(string)
			if !ok {
				return info.Error("list elements must be strings to be trimmed")
			}
			t = strings.Trim(t, cutset)
			list[i] = NewNode(t, binding)
		}
		result = list
	default:
		return info.Error("trim accepts only a string or list")
	}

	return result, info, true
}
