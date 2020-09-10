package dynaml

import (
	"fmt"
	"net"
)

type DivisionExpr struct {
	A Expression
	B Expression
}

func (e DivisionExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true

	a, info, ok := ResolveExpressionOrPushEvaluation(&e.A, &resolved, nil, binding, false)
	if !ok {
		return nil, info, false
	}

	b, info, ok := ResolveExpressionOrPushEvaluation(&e.B, &resolved, &info, binding, false)
	if !ok {
		return nil, info, false
	}

	if !resolved {
		return e, info, true
	}

	str, ok := a.(string)
	if ok {
		ip, cidr, err := net.ParseCIDR(str)
		if err != nil {
			return info.Error("first argument of division must be CIDR or number: %s", err)
		}
		ones, bits := cidr.Mask.Size()
		ip = ip.Mask(cidr.Mask)
		round := false

		bint, ok := b.(int64)
		if !ok {
			return info.Error("IP address division requires an integer argument")
		}
		if bint < 1 {
			return info.Error("IP address division requires a positive integer argument")
		}

		for bint > 1 {
			if bint%2 == 1 {
				round = true
			}
			bint = bint / 2
			ones++
		}
		if round {
			ones++
		}
		if ones > 32 {
			return info.Error("divisor too large for CIDR network size")
		}
		return (&net.IPNet{ip, net.CIDRMask(ones, bits)}).String(), info, true
	}

	a, b, err := NumberOperands(a, b)
	if err != nil {
		return info.Error("non-CIDR division requires number arguments")
	}
	if ib, ok := b.(int64); ok {
		if ib == 0 {
			return info.Error("division by zero")
		}
		return a.(int64) / ib, info, true
	}
	if b.(float64) == 0.0 {
		return info.Error("division by zero")
	}
	return a.(float64) / b.(float64), info, true
}

func (e DivisionExpr) String() string {
	return fmt.Sprintf("%s / %s", e.A, e.B)
}
