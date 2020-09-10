package dynaml

import "fmt"

type ListExpansion interface {
	Expression
	IsListExpansion() bool
}

type ListExpansionExpr struct {
	Expression
}

func (e ListExpansionExpr) String() string {
	return fmt.Sprintf("%s...", e.Expression)
}

func (e ListExpansionExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	return e.Expression.Evaluate(binding, locally)
}

func (e ListExpansionExpr) IsListExpansion() bool {
	return true
}

func IsListExpansion(e Expression) bool {
	va, ok := e.(ListExpansion)
	return ok && va.IsListExpansion()
}

func KeepArgWrapper(e Expression, orig Expression) Expression {
	if va, ok := orig.(ListExpansion); ok && va.IsListExpansion() {
		if _, ok := e.(ListExpansion); !ok {
			return ListExpansionExpr{e}
		}
	}
	if na, ok := orig.(NameArgument); ok {
		return NameArgument{na.Name, e}
	}
	return e
}
