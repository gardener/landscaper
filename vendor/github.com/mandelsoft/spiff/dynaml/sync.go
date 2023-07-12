package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
	"time"
)

type SyncExpr struct {
	A        Expression
	Cond     Expression
	Value    Expression
	Timeout  Expression
	function bool
	first    time.Time
	last     time.Time
}

func (e SyncExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true
	var value interface{}
	var info EvaluationInfo

	if e.first.IsZero() {
		e.first = time.Now()
	}

	errmsg := ""
	timeout := 5 * time.Minute
	if e.Timeout != nil {
		if _, ok := e.Timeout.(DefaultExpr); !ok {
			t, infot, ok := ResolveIntegerExpressionOrPushEvaluation(&e.Timeout, &resolved, nil, binding, false)
			if !ok {
				return nil, infot, ok
			}
			if !resolved {
				return e, info, true
			}
			timeout = time.Second * time.Duration(t)
		}
	}

	result := map[string]yaml.Node{}

	expr := e.A
	debug.Debug("sync value: %s\n", expr)
	value, infoe, ok := ResolveExpressionOrPushEvaluation(&expr, &resolved, nil, binding, false)

	if !ok {
		errmsg = infoe.Issue.Issue
		debug.Debug("sync arg failed: %s\n", errmsg)
		result[CATCH_VALID] = NewNode(false, binding)
		result[CATCH_ERROR] = NewNode(errmsg, binding)
	} else {
		if !resolved {
			return e, info, true
		}
		debug.Debug("sync arg succeeded\n")
		result[CATCH_VALID] = NewNode(true, binding)
		result[CATCH_ERROR] = NewNode("", binding)
		result[CATCH_VALUE] = NewNode(value, binding)
	}

	condbinding := binding
	if e.function {
		condbinding = binding.WithLocalScope(result)
	}

	inline := isInline(e.Cond) && !e.function
	cond, infoc, ok := e.Cond.Evaluate(condbinding, false)
	if !ok {
		return info.AnnotateError(infoc, "condition evaluation failed)")
	}

	if lambda, ok := cond.(LambdaValue); ok {
		args := []interface{}{}
		if result[CATCH_VALUE] != nil {
			args = append(args, result[CATCH_VALUE].Value())
		} else {
			args = append(args, nil)
		}
		switch len(lambda.lambda.Parameters) {
		case 1:
		case 2:
			debug.Debug("setting 2nd condition arg to error: %s\n", result[CATCH_ERROR].Value())
			args = append(args, result[CATCH_ERROR].Value())
		default:
			return info.Error("lambda expression for sync condition must take one or two arguments, found %d", len(lambda.lambda.Parameters))
		}
		resolved, result, sub, ok := lambda.Evaluate(inline, false, false, nil, args, binding, locally)
		if !resolved {
			return e, sub, ok
		}
		cond = result
	} else {
		if !e.function {
			return info.Error("sync condition must evaluate to a lambda value")
		}
	}

	switch v := cond.(type) {
	case bool:
		if !v {
			e.last = time.Now()

			if e.last.Before(e.first.Add(timeout)) {
				debug.Debug("sync failed but timeout not reached -> try again\n")
				return e, infoe, true
			}
			if errmsg != "" {
				debug.Debug("sync condition is finally false, err: %s\n", infoe.Issue.Issue)
				return nil, infoe, false
			} else {
				debug.Debug("sync condition is finally false\n")
				return info.Error("sync timeout reached")
			}
		} else {
			debug.Debug("sync condition is true\n")

		}
	case Expression:
		return e, info, true
	default:
		return info.Error("condition must evaluate to bool")
	}

	if !isDefaulted(e.Value) {
		debug.Debug("evaluating sync value\n")
		inline = isInline(e.Value) && !e.function
		value, infov, ok := e.Value.Evaluate(binding.WithLocalScope(result), false)
		if !ok {
			return info.AnnotateError(infoc, "value expression failed)")
		}
		if IsExpression(value) {
			return e, infov, true
		}

		if lambda, ok := value.(LambdaValue); ok && !e.function {
			args := []interface{}{}
			if result[CATCH_VALUE] != nil {
				debug.Debug("setting first value arg to value: %v\n", result[CATCH_VALUE].Value())
				args = append(args, result[CATCH_VALUE].Value())
			} else {
				debug.Debug("setting first value arg to nil\n")
				args = append(args, nil)
			}

			switch len(lambda.lambda.Parameters) {
			case 1:
			case 2:
				debug.Debug("setting 2nd value arg to error: %s\n", result[CATCH_ERROR].Value())
				args = append(args, result[CATCH_ERROR].Value())
			default:
				return info.Error("lambda expression for sync value must take one or two arguments, found %d", len(lambda.lambda.Parameters))
			}
			resolved, result, sub, ok := lambda.Evaluate(inline, false, false, nil, args, binding, locally)
			if !resolved {
				return e, sub, ok
			}
			return result, sub, ok
		} else {
			if !e.function {
				return info.Error("sync value expression must evaluate to lambda expression")
			}
		}
		return value, infov, ok
	}
	debug.Debug("returning sync value\n")
	return value, infoe, ok
}

func (e SyncExpr) String() string {
	return fmt.Sprintf("sync(%s)", e.A)
}

func (e CallExpr) sync(binding Binding) (interface{}, EvaluationInfo, bool) {
	var info EvaluationInfo
	switch len(e.Arguments) {
	case 2:
		return &SyncExpr{A: e.Arguments[0], Cond: e.Arguments[1], function: true}, info, true
	case 3:
		return &SyncExpr{A: e.Arguments[0], Cond: e.Arguments[1], Value: e.Arguments[2], function: true}, info, true
	case 4:
		return &SyncExpr{A: e.Arguments[0], Cond: e.Arguments[1], Value: e.Arguments[2], Timeout: e.Arguments[3], function: true}, info, true
	default:
		return info.Error("2 or 3 arguments required for sync")
	}
}
