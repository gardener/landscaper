package dynaml

import (
	"fmt"
)

type PreferExpr struct {
	expression Expression
}

func (e PreferExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {

	val, info, ok := e.expression.Evaluate(binding, locally)
	info.Preferred = true
	return val, info, ok
}

func (e PreferExpr) String() string {
	return fmt.Sprintf("prefer %s", e.expression)
}
