package dynaml

import (
	"fmt"
)

type BooleanExpr struct {
	Value bool
}

func (e BooleanExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	return e.Value, DefaultInfo(), true
}

func (e BooleanExpr) String() string {
	return fmt.Sprintf("%v", e.Value)
}
