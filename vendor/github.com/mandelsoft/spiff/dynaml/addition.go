package dynaml

import (
	"fmt"
	"net"
	"reflect"
)

type AdditionExpr struct {
	A Expression
	B Expression
}

func (e AdditionExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
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
		ip := net.ParseIP(str)
		if ip == nil {
			return info.Error("first argument for addition must be IP address or number")
		}
		bint, ok := b.(int64)
		if !ok {
			return info.Error("addition argument for an IP address requires an integer argument")
		}
		return IPAdd(ip, bint).String(), info, true
	}
	a, b, err := NumberOperands(a, b)
	if err != nil {
		return info.Error("non-IP address addition requires number arguments")
	}
	if _, ok := a.(int64); ok {
		return a.(int64) + b.(int64), info, true
	}
	return a.(float64) + b.(float64), info, true
}

func (e AdditionExpr) String() string {
	return fmt.Sprintf("%s + %s", e.A, e.B)
}

func IPAdd(ip net.IP, offset int64) net.IP {
	for j := len(ip) - 1; j >= 0; j-- {
		tmp := offset + int64(ip[j])
		ip[j] = byte(tmp)
		if tmp < 0 {
			tmp = tmp - 256
		}
		offset = tmp / 256
		if offset == 0 {
			break
		}
	}
	return ip
}

func NumberOperands(a, b interface{}) (interface{}, interface{}, error) {
	ia, iaok := a.(int64)
	fa, faok := a.(float64)
	if !iaok && !faok {
		return nil, nil, fmt.Errorf("operand must be integer or float (%s)", reflect.TypeOf(a))
	}
	ib, ibok := b.(int64)
	fb, fbok := b.(float64)
	if !ibok && !fbok {
		return nil, nil, fmt.Errorf("operand must be integer or float (%s)", reflect.TypeOf(b))
	}
	if iaok == ibok {
		return a, b, nil
	}
	if faok {
		return fa, float64(ib), nil
	}
	return float64(ia), fb, nil
}
