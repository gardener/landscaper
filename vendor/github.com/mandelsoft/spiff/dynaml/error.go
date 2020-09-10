package dynaml

func func_error(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	n, info, ok := format("error", arguments, binding)
	if !ok {
		return n, info, ok
	}
	return info.Error("%s", n)
}
