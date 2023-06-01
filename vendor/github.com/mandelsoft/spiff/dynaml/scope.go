package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type ScopeExpr struct {
	CreateMapExpr
	E Expression
}

func (e ScopeExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	local, info, ok := e.CreateMapExpr.Evaluate(binding, locally)

	if !ok {
		return local, info, ok
	}

	if _, ok := local.(Expression); ok {
		return e, info, ok
	}
	b := local.(map[string]yaml.Node)
	binding = binding.WithLocalScope(b)
	debug.Debug("SCOPE: %s\n", binding)
	local, info, ok = e.E.Evaluate(binding, locally)
	if ok && IsExpression(local) {
		return e, info, ok
	}
	return local, info, ok
}

func (e ScopeExpr) String() string {
	result := "("
	sep := ""
	for _, a := range e.Assignments {
		result += sep + a.String()
		sep = ", "
	}
	return fmt.Sprintf("%s ) %s", result, e.E)
}
