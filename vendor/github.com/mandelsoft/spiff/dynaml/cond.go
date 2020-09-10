package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/yaml"
)

type CondExpr struct {
	C Expression
	T Expression
	F Expression
}

func (e CondExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true
	info := DefaultInfo()
	var infoc EvaluationInfo
	var result interface{}

	a, info, ok := ResolveExpressionOrPushEvaluation(&e.C, &resolved, &info, binding, false)
	if !ok {
		return nil, info, false
	}
	if !resolved {
		return e, info, true
	}
	if toBool(a) {
		result, infoc, ok = e.T.Evaluate(binding, false)
	} else {
		result, infoc, ok = e.F.Evaluate(binding, false)
	}
	return result, infoc.Join(info), ok
}

func (e CondExpr) String() string {
	return fmt.Sprintf("%s ? %s : %s", e.C, e.T, e.F)
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}

	switch eff := v.(type) {
	case bool:
		return eff
	case string:
		return len(eff) > 0
	case int64:
		return eff != 0
	case []yaml.Node:
		return len(eff) != 0
	case map[string]yaml.Node:
		return len(eff) != 0
	default:
		return true
	}
}
