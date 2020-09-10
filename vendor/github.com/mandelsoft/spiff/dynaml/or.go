package dynaml

import (
	"fmt"
	"reflect"
)

type OrExpr struct {
	A Expression
	B Expression
}

func (e OrExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	a, infoa, ok := e.A.Evaluate(binding, false)
	if ok && !infoa.Undefined {
		if reflect.DeepEqual(a, e.A) {
			//fmt.Printf("================== %s\n", e.A)
			return e, infoa, true
		}
		//fmt.Printf("++++++++++++++++++ %s\n", e.A)
		if isExpression(a) {
			return e, infoa, true
		}
		return a, infoa, true
	}
	//fmt.Printf("------------------ %t %t %s\n", ok, infoa.Undefined, e.A)
	b, infob, ok := e.B.Evaluate(binding, false)
	info := infoa.Join(infob)
	info.Undefined = infob.Undefined
	return b, info, ok
}

func (e OrExpr) String() string {
	return fmt.Sprintf("%s || %s", e.A, e.B)
}
