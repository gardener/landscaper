package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
	"strconv"
)

func func_makemap(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	result := make(map[string]yaml.Node)
	var key string

	if len(arguments) == 1 {
		list, ok := arguments[0].([]yaml.Node)
		if !ok {
			return info.Error("single argument flavor of mapentry requires a list")
		}
		for i, elem := range list {
			m, ok := elem.Value().(map[string]yaml.Node)
			if !ok {
				return info.Error("entry %d is no map entry", i)
			}
			k, ok := m["key"]
			if !ok {
				return info.Error("entry %d has no entry 'key'", i)
			}
			switch elem := k.Value().(type) {
			case string:
				key = elem
			case int64:
				key = strconv.FormatInt(elem, 10)
			case bool:
				key = strconv.FormatBool(elem)
			default:
				return info.Error("invalid type for 'key' value of entry %d", i)
			}
			v, ok := m["value"]
			if !ok {
				return info.Error("entry %d has no entry 'value'", i)
			}
			result[key] = v
		}
	} else if len(arguments)%2 == 0 {
		for i := 0; i < len(arguments); i += 2 {
			switch elem := arguments[i].(type) {
			case string:
				key = elem
			case int64:
				key = strconv.FormatInt(elem, 10)
			case bool:
				key = strconv.FormatBool(elem)
			default:
				return info.Error("invalid type for key value of arument pair %d", i/2)
			}

			result[key] = NewNode(arguments[i+1], binding)
		}
	} else {
		return info.Error("mapentry takes one or two arguments")
	}

	return result, info, true
}
