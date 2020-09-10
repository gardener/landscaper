package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

func func_length(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var result interface{}
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("length takes exactly 1 argument")
	}

	switch v := arguments[0].(type) {
	case []yaml.Node:
		result = len(v)
	case map[string]yaml.Node:
		result = len(v)
	case string:
		result = len(v)
	default:
		return info.Error("invalid type for function length")
	}
	return yaml.MassageType(result), info, true
}
