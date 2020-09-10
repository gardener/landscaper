package dynaml

import (
	"fmt"
)

type ModuloExpr struct {
	A Expression
	B Expression
}

func (e ModuloExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true

	aint, info, ok := ResolveIntegerExpressionOrPushEvaluation(&e.A, &resolved, nil, binding, false)
	if !ok {
		return nil, info, false
	}

	bint, info, ok := ResolveIntegerExpressionOrPushEvaluation(&e.B, &resolved, &info, binding, false)
	if !ok {
		return nil, info, false
	}

	if !resolved {
		return e, info, true
	}

	if bint == 0 {
		return info.Error("division by zero")
	}
	return aint % bint, info, true
}

func (e ModuloExpr) String() string {
	return fmt.Sprintf("%s %% %s", e.A, e.B)
}
