package dynaml

import (
	"fmt"
	"net"
)

type MultiplicationExpr struct {
	A Expression
	B Expression
}

func (e MultiplicationExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
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
			return info.Error("first argument of multiplication must be CIDR or number: %s", err)
		}
		ones, _ := cidr.Mask.Size()
		size := int64(1 << (32 - uint32(ones)))

		bint, ok := b.(int64)
		if !ok {
			return info.Error("CIDR multiplication requires an integer argument")
		}

		ip = IPAdd(ip.Mask(cidr.Mask), size*bint)
		return (&net.IPNet{ip, cidr.Mask}).String(), info, true
	}

	a, b, err := NumberOperands(a, b)
	if err != nil {
		return info.Error("non-CIDR multiplication requires number arguments")
	}
	if _, ok := a.(int64); ok {
		return a.(int64) * b.(int64), info, true
	}
	return a.(float64) * b.(float64), info, true
}

func (e MultiplicationExpr) String() string {
	return fmt.Sprintf("%s * %s", e.A, e.B)
}
