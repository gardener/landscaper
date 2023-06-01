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
		var cidr *net.IPNet
		var err error
		ip := net.ParseIP(str)
		if ip == nil {
			ip, cidr, err = net.ParseCIDR(str)
			if err != nil {
				return info.Error("first argument for addition must be IP address, CIDR or number")
			}
		}
		bint, ok := b.(int64)
		if !ok {
			return info.Error("addition argument for an IP address requires an integer argument")
		}
		ip = IPAdd(ip, bint)
		if cidr != nil {
			if !cidr.Contains(ip) {
				return info.Error("resulting ip address not in CIDR range")
			}
			cidr.IP = ip
			return cidr.String(), info, true
		}
		return ip.String(), info, true
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

const TYPE_INT = 1
const TYPE_FLOAT = 2
const TYPE_NUMBER = TYPE_FLOAT | TYPE_INT

func NumberOperandsN(convert int, ops ...interface{}) ([]interface{}, bool, error) {
	isInt := true
	var r []interface{}

	for n, o := range ops {
		v, ok := o.(int64)
		if ok {
			if isInt && (convert&TYPE_INT != 0) {
				r = append(r, v)
			} else {
				r = append(r, float64(v))
			}
		} else {
			v, ok := o.(float64)
			if ok {
				if isInt {
					isInt = false
					if convert == TYPE_NUMBER {
						for i, v := range r {
							r[i] = float64(v.(int64))
						}
					}
				}
				if convert&TYPE_FLOAT != 0 {
					r = append(r, v)
				} else {
					r = append(r, int64(v))
				}
			} else {
				return nil, false, fmt.Errorf("operand %d must be integer or float (%s)", n, reflect.TypeOf(o))
			}
		}
	}
	return r, isInt, nil
}
