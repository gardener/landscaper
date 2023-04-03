package dynaml

import (
	"fmt"
	"net"
)

type SubtractionExpr struct {
	A Expression
	B Expression
}

func (e SubtractionExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
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
		var cidr *net.IPNet
		var err error
		ip := net.ParseIP(str)
		if ip == nil {
			ip, cidr, err = net.ParseCIDR(str)
			if err != nil {
				return info.Error("first argument for substraction must be IP address, CIDR or number")
			}
		}
		bint, bok := b.(int64)
		if bok {
			ip = IPAdd(ip, -bint)
			if cidr != nil {
				if !cidr.Contains(ip) {
					return info.Error("resulting ip address not in CIDR range")
				}
				cidr.IP = ip
				return cidr.String(), info, true
			}
			return ip.String(), info, true
		}
		bstr, ok := b.(string)
		if ok {
			ipb := net.ParseIP(bstr)
			if ipb == nil {
				ip, _, err := net.ParseCIDR(bstr)
				if err != nil {
					return info.Error("second argument for substraction must be IP address, CIDR or number")
				}
				ipb = ip
			}
			if len(ip) != len(ipb) {
				return info.Error("IP type mismatch (%d != %d)", len(ip), len(ipb))
			}
			return DiffIP(ip, ipb), info, true
		}
		return info.Error("second argument of IP address subtraction must be IP address or integer")
	}

	a, b, err := NumberOperands(a, b)
	if err != nil {
		return info.Error("non-IP address subtration requires number arguments")
	}
	if _, ok := a.(int64); ok {
		return a.(int64) - b.(int64), info, true
	}
	return a.(float64) - b.(float64), info, true
}

func (e SubtractionExpr) String() string {
	return fmt.Sprintf("%s - %s", e.A, e.B)
}
