package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type SumExpr struct {
	A      Expression
	I      Expression
	Lambda Expression
}

func (e SumExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true

	debug.Debug("evaluate sum")
	value, info, ok := ResolveExpressionOrPushEvaluation(&e.A, &resolved, nil, binding, true)
	if !ok {
		return nil, info, false
	}
	initial, info, ok := ResolveExpressionOrPushEvaluation(&e.I, &resolved, &info, binding, false)
	if !ok {
		return nil, info, false
	}
	undef := info.Undefined // TODO
	debug.Debug("sum: initial undef: %t", undef)
	inline := isInline(e.Lambda)
	lvalue, infoe, ok := ResolveExpressionOrPushEvaluation(&e.Lambda, &resolved, nil, binding, false)
	if !ok {
		return nil, infoe, false
	}

	if !resolved {
		return e, info.Join(infoe), ok
	}

	lambda, ok := lvalue.(LambdaValue)
	if !ok {
		return infoe.Error("sum requires a lambda value")
	}

	debug.Debug("map: using lambda %+v\n", lambda)
	var result interface{}
	switch value.(type) {
	case []yaml.Node:
		resolved, result, info, ok = sumList(inline, value.([]yaml.Node), lambda, initial, binding)

	case map[string]yaml.Node:
		resolved, result, info, ok = sumMap(inline, value.(map[string]yaml.Node), lambda, initial, binding)

	default:
		return info.Error("map or list required for sum")
	}
	if !ok {
		return nil, info, false
	}
	if !resolved {
		return e, info, true
	}
	debug.Debug("sum: --> %+v\n", result)
	return result, info, true
}

func (e SumExpr) String() string {
	lambda, ok := e.Lambda.(LambdaExpr)
	if ok {
		return fmt.Sprintf("sum[%s|%s%s]", e.A, e.I, fmt.Sprintf("%s", lambda)[len("lambda"):])
	} else {
		return fmt.Sprintf("sum[%s|%s|%s]", e.A, e.I, e.Lambda)
	}
}

func sumList(inline bool, source []yaml.Node, e LambdaValue, initial interface{}, binding Binding) (bool, interface{}, EvaluationInfo, bool) {
	inp := make([]interface{}, len(e.lambda.Parameters))
	result := initial
	info := DefaultInfo()

	if len(e.lambda.Parameters) > 3 {
		info.SetError("mapping expression take a maximum of 3 arguments")
		return true, nil, info, false
	}
	if len(e.lambda.Parameters) < 2 {
		info.SetError("mapping expression take a minimum of 2 arguments")
		return true, nil, info, false
	}
	debug.Debug("sum:  initial: %+v\n", initial)
	for i, n := range source {
		debug.Debug("sum:  mapping for %d: %+v\n", i, n)
		inp[0] = result
		inp[1] = i
		inp[len(inp)-1] = n.Value()
		resolved, mapped, info, ok := e.Evaluate(inline, false, false, nil, inp, binding, false)
		if !ok {
			debug.Debug("sum:  %d %+v: failed\n", i, n)
			return true, nil, info, false
		}
		if !resolved {
			return false, nil, info, ok
		}
		_, ok = mapped.(Expression)
		if ok {
			debug.Debug("sum:  %d unresolved  -> KEEP\n", i)
			return false, nil, info, true
		}
		debug.Debug("sum:  %d --> %+v\n", i, mapped)
		result = mapped
	}
	debug.Debug("sum:  result: %+v\n", result)
	return true, result, info, true
}

func sumMap(inline bool, source map[string]yaml.Node, e LambdaValue, initial interface{}, binding Binding) (bool, interface{}, EvaluationInfo, bool) {
	inp := make([]interface{}, len(e.lambda.Parameters))
	result := initial
	info := DefaultInfo()

	keys := getSortedKeys(source)
	for _, k := range keys {
		n := source[k]
		debug.Debug("map:  mapping for %s: %+v\n", k, n)
		inp[0] = result
		inp[1] = k
		inp[len(inp)-1] = n.Value()
		resolved, mapped, info, ok := e.Evaluate(inline, false, false, nil, inp, binding, false)
		if !ok {
			debug.Debug("map:  %s %+v: failed\n", k, n)
			return true, nil, info, false
		}
		if !resolved {
			return false, nil, info, ok
		}
		_, ok = mapped.(Expression)
		if ok {
			debug.Debug("map:  %s unresolved  -> KEEP\n", k)
			return false, nil, info, true
		}
		debug.Debug("map:  %s --> %+v\n", k, mapped)
		result = mapped
	}
	return true, result, info, true
}
