package dynaml

func (e CallExpr) valid(binding Binding) (interface{}, EvaluationInfo, bool) {
	pushed := make([]Expression, len(e.Arguments))
	ok := true
	resolved := true
	valid := true
	var val interface{}

	copy(pushed, e.Arguments)
	for i, _ := range pushed {
		val, _, ok = ResolveExpressionOrPushEvaluation(&pushed[i], &resolved, nil, binding, true)
		if resolved && !ok {
			return false, DefaultInfo(), true
		}
		valid = valid && (val != nil)
	}
	if !resolved {
		return e, DefaultInfo(), true
	}
	return valid, DefaultInfo(), ok
}

func (e CallExpr) defined(binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	pushed := make([]Expression, len(e.Arguments))
	ok := true
	resolved := true

	copy(pushed, e.Arguments)
	for i, _ := range pushed {
		_, info, ok = ResolveExpressionOrPushEvaluation(&pushed[i], &resolved, nil, binding, true)
		if resolved {
			if !ok {
				return false, DefaultInfo(), true
			}
			if info.Undefined {
				return false, DefaultInfo(), true
			}
		}
	}
	if !resolved {
		return e, DefaultInfo(), true
	}
	return true, DefaultInfo(), ok
}
