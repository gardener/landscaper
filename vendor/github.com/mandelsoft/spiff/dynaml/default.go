package dynaml

type DefaultExpr struct {
}

func (e DefaultExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	return info.Error("OOPS: default value not handled")
}

func (e DefaultExpr) String() string {
	return ""
}

func isDefaulted(e Expression) bool {
	if e == nil {
		return true
	}
	_, ok := e.(DefaultExpr)
	return ok
}
