package dynaml

import (
	"strings"

	"github.com/mandelsoft/spiff/legacy/candiedyaml"
	"github.com/mandelsoft/spiff/yaml"
)

func func_as_json(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("asjson takes exactly one argument")
	}

	result, err := yaml.ValueToJSON(arguments[0])
	if err != nil {
		return info.Error("cannot jsonencode: %s", err)
	}
	return string(result), info, true
}

func func_as_yaml(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("asyaml takes exactly one argument")
	}

	result, err := candiedyaml.Marshal(arguments[0])
	if err != nil {
		return info.Error("cannot yamlencode: %s", err)
	}
	return string(result), info, true
}

func func_parse_yaml(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("parse takes one or two arguments")
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for parse must be a string")
	}

	mode := "import"
	if len(arguments) > 1 {
		mode, ok = arguments[1].(string)
		if !ok {
			return info.Error("second argument for parse must be a string")
		}
	}

	name := strings.Join(binding.Path(), ".")
	return ParseData(name, []byte(str), mode, binding)
}
