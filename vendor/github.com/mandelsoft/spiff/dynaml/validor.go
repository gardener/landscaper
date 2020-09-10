package dynaml

import (
	"fmt"
	"reflect"
)

type ValidOrExpr struct {
	A Expression
	B Expression
}

func (e ValidOrExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	a, infoa, ok := e.A.Evaluate(binding, false)
	if ok {
		if reflect.DeepEqual(a, e.A) {
			return nil, infoa, false
		}
		if isExpression(a) {
			return e, infoa, true
		}
		if a != nil {
			return a, infoa, true
		}
	}

	b, infob, ok := e.B.Evaluate(binding, false)
	info := infoa.Join(infob)
	info.Undefined = infob.Undefined
	return b, info, ok
}

func (e ValidOrExpr) String() string {
	return fmt.Sprintf("%s // %s", e.A, e.B)
}
