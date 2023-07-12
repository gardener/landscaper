package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/debug"
)

const (
	OpOr = "-or"
)

type LogOrExpr struct {
	A Expression
	B Expression
}

func (e LogOrExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	a, b, info, resolved, first_ok, all_ok := resolveLOperands(e.A, e.B, binding)
	if !first_ok {
		debug.Debug("OR: failed %#v, %#v\n", e.A, e.B)
		return nil, info, false
	}
	if !resolved {
		return e, info, true
	}
	debug.Debug("OR: %#v, %#v\n", a, b)
	inta, ok := a.(int64)
	if ok {
		if !all_ok {
			return nil, info, false
		}
		return inta | b.(int64), info, true
	}
	if toBool(a) {
		return true, info, true
	}
	if !all_ok {
		return nil, info, false
	}
	return toBool(b), info, true
}

func (e LogOrExpr) String() string {
	return fmt.Sprintf("%s %s %s", e.A, OpOr, e.B)
}

func resolveLOperands(a, b Expression, binding Binding) (eff_a, eff_b interface{}, info EvaluationInfo, resolved bool, first_ok bool, ok bool) {
	var va, vb interface{}
	var infoa, infob EvaluationInfo

	va, infoa, first_ok = a.Evaluate(binding, false)
	if first_ok {
		if IsExpression(va) {
			return nil, nil, infoa, false, true, true
		}

		vb, infob, ok = b.Evaluate(binding, false)
		info = infoa.Join(infob)
		if !ok {
			return va, nil, info, true, true, false
		}

		if IsExpression(vb) {
			return nil, nil, info, false, true, true
		}

		resolved = true
		eff_a, ok = va.(int64)
		if ok {
			eff_b, ok = vb.(int64)
			if ok {
				return
			}
		}
		return toBool(va), toBool(vb), info, true, true, true
	}

	return nil, nil, info, false, false, false
}
