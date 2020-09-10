package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/debug"
)

type NotExpr struct {
	Expr Expression
}

func (e NotExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true
	v, info, ok := ResolveExpressionOrPushEvaluation(&e.Expr, &resolved, nil, binding, false)
	if !ok {
		return nil, info, false
	}
	if !resolved {
		return e, info, true
	}

	debug.Debug("NOT: %#v\n", v)
	return !toBool(v), info, true
}

func (e NotExpr) String() string {
	return fmt.Sprintf("!(%s)", e.Expr)
}
