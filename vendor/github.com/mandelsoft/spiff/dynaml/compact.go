package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	RegisterFunction("compact", func_compact)
}

func func_compact(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("compact takes exactly 1 argument")
	}

	list, ok := arguments[0].([]yaml.Node)
	if !ok {
		return info.Error("invalid type for function compact")
	}

	var newList []yaml.Node

	for _, v := range list {
		found := true

		if v != nil && v.Value() != nil {
			switch elem := v.Value().(type) {
			case string:
				found = len(elem) > 0
			case int64:
			case map[string]yaml.Node:
				found = len(elem) > 0
			case []yaml.Node:
				found = len(elem) > 0
			}
			if found {
				newList = append(newList, v)
			}
		}
	}
	return newList, info, true
}
