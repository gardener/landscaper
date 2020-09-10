package dynaml

type UndefinedExpr struct{}

func (e UndefinedExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	info.Undefined = true
	return nil, info, true
}

func (e UndefinedExpr) String() string {
	return "~~"
}
