package dynaml

type NilExpr struct{}

func (e NilExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	return nil, DefaultInfo(), true
}

func (e NilExpr) String() string {
	return "nil"
}
