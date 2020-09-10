package dynaml

import (
	"fmt"
)

type GroupedExpr struct {
	Expr Expression
}

func (e GroupedExpr) String() string {
	return fmt.Sprintf("( %s )", e.Expr)
}

func (e GroupedExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	return e.Expr.Evaluate(binding, locally)
}
