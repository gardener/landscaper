package dynaml

import "strings"

func func_lower(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return _modifystring("lower", strings.ToLower, arguments, binding)
}

func func_upper(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return _modifystring("upper", strings.ToUpper, arguments, binding)
}

func _modifystring(name string, mod func(string) string, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {

	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("%s requires one argument", name)
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for %s must be a string", name)
	}

	return mod(str), info, true
}
