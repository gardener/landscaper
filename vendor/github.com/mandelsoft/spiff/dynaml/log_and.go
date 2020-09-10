package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/debug"
)

const (
	OpAnd = "-and"
)

type LogAndExpr struct {
	A Expression
	B Expression
}

func (e LogAndExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	a, b, info, resolved, first_ok, all_ok := resolveLOperands(e.A, e.B, binding)
	if !first_ok {
		return nil, info, false
	}
	if !resolved {
		return e, info, true
	}
	debug.Debug("AND: %#v, %#v\n", a, b)
	inta, ok := a.(int64)
	if ok {
		if !all_ok {
			return nil, info, false
		}
		return inta & b.(int64), info, true
	}
	if !toBool(a) {
		return false, info, true
	}
	if !all_ok {
		return nil, info, false
	}
	return toBool(b), info, true
}

func (e LogAndExpr) String() string {
	return fmt.Sprintf("%s %s %s", e.A, OpAnd, e.B)
}
