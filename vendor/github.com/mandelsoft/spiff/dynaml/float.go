package dynaml

import (
	"strconv"
)

type FloatExpr struct {
	Value float64
}

func (e FloatExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	return e.Value, DefaultInfo(), true
}

func (e FloatExpr) String() string {
	return strconv.FormatFloat(e.Value, 'g', -1, 64)
}
