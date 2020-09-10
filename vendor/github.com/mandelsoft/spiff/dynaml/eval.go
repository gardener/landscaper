package dynaml

func func_eval(arguments []interface{}, binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("one argument required for 'eval'")
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("string argument required for 'eval'")
	}

	expr, err := Parse(str, binding.Path(), binding.StubPath())
	if err != nil {
		return info.Error("(%s)\t %s", str, err)
	}
	return expr.Evaluate(binding, locally)
}
