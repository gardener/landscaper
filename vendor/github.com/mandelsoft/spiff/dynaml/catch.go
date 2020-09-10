package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

const CATCH_ERROR = "error"
const CATCH_VALUE = "value"
const CATCH_VALID = "valid"

type CatchExpr struct {
	A      Expression
	Lambda Expression
}

func (e CatchExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true
	var value interface{}
	var info EvaluationInfo
	var lambda *LambdaValue
	inline := isInline(e.Lambda)

	if e.Lambda != nil {
		debug.Debug("catch EXPR with lambda\n")
		lvalue, info, ok := ResolveExpressionOrPushEvaluation(&e.Lambda, &resolved, nil, binding, false)
		if !ok {
			return nil, info, false
		}

		if !resolved {
			return e, info.Join(info), ok
		}

		l, ok := lvalue.(LambdaValue)
		if !ok {
			return info.Error("catch requires a lambda value")
		}

		if len(l.lambda.Parameters) > 2 {
			return info.Error("catch function takes a maximum of 2 arguments")
		}

		lambda = &l
	}

	value, infoe, ok := ResolveExpressionOrPushEvaluation(&e.A, &resolved, nil, binding, false)
	debug.Debug("catch %t resolved: %t, err=%s, %v\n", ok, resolved, infoe.Issue.Issue, value)
	if !resolved && ok {
		return e, infoe, true
	}
	// no lambda -> returning deprecated  map
	if lambda == nil {
		result := map[string]yaml.Node{}
		if !ok {
			debug.Debug("catch arg failed\n")
			result[CATCH_VALID] = NewNode(false, binding)
			result[CATCH_ERROR] = NewNode(infoe.Issue.Issue, binding)
			return result, info, true
		}

		info.Join(infoe)
		if !resolved {
			return e, info, true
		}

		debug.Debug("catch arg succeeded\n")
		result[CATCH_VALID] = NewNode(true, binding)
		result[CATCH_ERROR] = NewNode("", binding)
		result[CATCH_VALUE] = NewNode(value, binding)
		return result, info, ok
	}

	// using lambda value

	debug.Debug("catch using lambda: %s\n", lambda)
	inp := make([]interface{}, len(lambda.lambda.Parameters))
	if !ok {
		debug.Debug("catch failed: %s\n", infoe.Issue.Issue)
		value = nil
	} else {
		debug.Debug("catch succeeded: %v\n", value)
	}
	inp[0] = value
	switch len(lambda.lambda.Parameters) {
	case 1:
	case 2:
		if ok {
			debug.Debug("setting 2nd catch arg to nil\n")
			inp[1] = nil
		} else {
			debug.Debug("setting 2nd catch arg to error: %s\n", infoe.Issue.Issue)
			inp[1] = infoe.Issue.Issue
		}
	default:
		return info.Error("lambda expression for sync condition must take one or two arguments, found %d", len(lambda.lambda.Parameters))
	}

	resolved, mapped, info, ok := lambda.Evaluate(inline, false, false, nil, inp, binding, false)
	if !ok {
		debug.Debug("catch lambda failed\n")
		return nil, info, false
	}
	if !resolved {
		debug.Debug("catch: lambda unresolved  -> KEEP\n")
		return e, info, ok
	}
	_, ok = mapped.(Expression)
	if ok {
		debug.Debug("catch: returned expression -> KEEP\n")
		return e, info, true
	}
	debug.Debug("catch:  done: %#v\n", mapped)
	return mapped, info, true
}

func (e CatchExpr) String() string {
	if e.Lambda == nil {
		return fmt.Sprintf("catch(%s)", e.A)
	} else {
		lambda, ok := e.Lambda.(LambdaExpr)
		if ok {
			return fmt.Sprintf("catch[%s%s]", e.A, fmt.Sprintf("%s", lambda)[len("lambda"):])
		} else {
			return fmt.Sprintf("catch[%s|%s]", e.A, e.Lambda)
		}
	}
}

func (e CallExpr) catch(binding Binding) (interface{}, EvaluationInfo, bool) {
	var info EvaluationInfo
	if len(e.Arguments) != 1 {
		return info.Error("catch requires a single argument")
	}
	return CatchExpr{e.Arguments[0], nil}, info, true
}
