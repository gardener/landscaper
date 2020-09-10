package dynaml

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type ComparisonExpr struct {
	A  Expression
	Op string
	B  Expression
}

func (e ComparisonExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
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

	var result bool
	var infor EvaluationInfo

	switch e.Op {
	case "==":
		result, infor, ok = compareEquals(a, b)
	case "!=":
		result, infor, ok = compareEquals(a, b)
		result = !result
	case "<=", "<", ">", ">=":
		switch va := a.(type) {
		case int64:
			vb, ok := b.(int64)
			if !ok {
				return infor.Error("comparision %s only for integers or strings", e.Op)
			}
			switch e.Op {
			case "<=":
				result = va <= vb
			case "<":
				result = va < vb
			case ">":
				result = va > vb
			case ">=":
				result = va >= vb
			}

		case string:
			vb, ok := b.(string)
			if !ok {
				return infor.Error("comparision %s only for strings or integers", e.Op)
			}
			switch e.Op {
			case "<=":
				result = va <= vb
			case "<":
				result = va < vb
			case ">":
				result = va > vb
			case ">=":
				result = va >= vb
			}
		}
	}
	infor = info.Join(infor)

	if !ok {
		return nil, infor, false
	}
	return result, infor, true
}

func (e ComparisonExpr) String() string {
	return fmt.Sprintf("%s %s %s", e.A, e.Op, e.B)
}

func compareEquals(a, b interface{}) (bool, EvaluationInfo, bool) {
	info := DefaultInfo()

	debug.Debug("compare a '%T'\n", a)
	debug.Debug("compare b '%T' \n", b)
	if a == nil {
		if b == nil {
			return true, info, true
		}
		debug.Debug("compare failed: nil vs non-nil '%v'\n", b)
		return false, info, false
	} else {
		if b == nil {
			debug.Debug("compare failed: nil vs non-nil '%v'\n", a)
			return false, info, false
		}
	}
	switch va := a.(type) {
	case string:
		var vb string
		switch v := b.(type) {
		case string:
			vb = v
		case int64:
			vb = strconv.FormatInt(v, 10)
		case LambdaValue:
			vb = v.String()
		case bool:
			vb = strconv.FormatBool(v)
		default:
			info.Issue = yaml.NewIssue("types uncomparable")
			return false, info, false
		}
		if va == vb {
			return true, info, true
		}
		debug.Debug("compare failed: %v != %v\n", va, vb)
		return false, info, true

	case int64:
		var vb int64
		var err error
		switch v := b.(type) {
		case string:
			vb, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				debug.Debug("compare failed: no int '%v'\n", v)
				return false, info, true
			}
		case int64:
			vb = v
		case bool:
			if v {
				vb = 1
			} else {
				vb = 0
			}
		default:
			debug.Debug("compare failed: no int '%T'\n", b)
			return false, info, true
		}
		if va == vb {
			return true, info, true
		}
		debug.Debug("compare failed: %v != %v\n", va, vb)
		return false, info, true

	case bool:
		var vb bool
		var err error
		switch v := b.(type) {
		case bool:
			vb = v
		case string:
			vb, err = strconv.ParseBool(strings.ToLower(v))
			if err != nil {
				debug.Debug("compare failed: no bool '%v'\n", v)
				return false, info, true
			}
		default:
			debug.Debug("compare failed: no bool '%T'\n", b)
			return false, info, true
		}
		if va == vb {
			return true, info, true
		}
		debug.Debug("compare failed: %v != %v\n", va, vb)
		return false, info, true

	case yaml.ComparableValue:
		if vb, ok := b.(yaml.ComparableValue); ok {
			return va.EquivalentTo(vb), info, true
		}

	case []yaml.Node:
		vb, ok := b.([]yaml.Node)
		if !ok || len(va) != len(vb) {
			debug.Debug("compare list len mismatch")
			return false, info, true
		}
		for i, v := range vb {
			result, info, _ := compareEquals(va[i].Value(), v.Value())
			if !result {
				debug.Debug("compare list entry %d mismatch\n", i)
				return false, info, true
			}
		}
		return true, info, true

	case map[string]yaml.Node:
		vb, ok := b.(map[string]yaml.Node)
		if !ok || len(va) != len(vb) {
			debug.Debug("compare map len mismatch")
			return false, info, true
		}

		for k, v := range vb {
			vaa, ok := va[k]
			if ok {
				result, info, _ := compareEquals(vaa.Value(), v.Value())
				if !result {
					debug.Debug("compare map entry %s mismatch\n", k)
					return false, info, true
				}
			} else {
				debug.Debug("compare map entry %s not found\n", k)
				return false, info, true
			}
		}
		return true, info, true

	}
	debug.Debug("compare failed\n")
	return false, info, true
}
