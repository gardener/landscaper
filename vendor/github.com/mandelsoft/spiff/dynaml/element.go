package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
	"strconv"
)

func func_element(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 2 {
		return info.Error("element takes exactly two arguments")
	}

	if arguments[0] == nil || arguments[1] == nil {
		return info.Error("function element does not take nil arguments")
	}
	switch data := arguments[0].(type) {
	case []yaml.Node:
		var index int64 = 0
		var err error
		switch v := arguments[1].(type) {
		case int64:
			index = v
		case string:
			index, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return info.Error("invalid value '%s' of index argument for function element", v)
			}
		default:
			return info.Error("invalid type of index argument for function element")
		}
		if index < 0 {
			return info.Error("index lower than 0")
		}
		if index >= int64(len(data)) {
			return info.Error("index greater or equal list size")
		}
		return data[index].Value(), info, true

	case map[string]yaml.Node:
		index, ok := arguments[1].(string)
		if !ok {
			return info.Error("map key (%v) must be of type string", arguments[1])
		}
		e, ok := data[index]
		if !ok {
			return info.Error("map key '%s' not found", index)
		}
		return e.Value(), info, true
	default:
		return info.Error("invalid type for first argument of function element")
	}
}
