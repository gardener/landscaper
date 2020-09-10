package dynaml

func (e CallExpr) require(binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(e.Arguments) != 1 {
		return info.Error("one argument expected for 'require'")
	}
	pushed := e.Arguments[0]
	ok := true
	resolved := true

	val, _, ok := ResolveExpressionOrPushEvaluation(&pushed, &resolved, nil, binding, true)
	if !resolved {
		return e, info, true
	}

	if !ok || val == nil {
		return info.Error("required expression %q undefined", e.Arguments[0])
	}

	return val, info, val != nil
}
