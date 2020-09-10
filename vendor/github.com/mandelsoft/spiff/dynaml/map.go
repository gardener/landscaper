package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/yaml"
)

type CreateMapExpr struct {
	Assignments []Assignment
}

func (e CreateMapExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {

	newMap := make(map[string]yaml.Node)
	info := DefaultInfo()
	resolved := true

	for _, a := range e.Assignments {
		key, info, ok := ResolveExpressionOrPushEvaluation(&a.Key, &resolved, &info, binding, locally)
		if !ok || !resolved {
			return e, info, ok
		}

		kstr, ok := key.(string)
		if !ok {
			return info.Error("assignment target must evaluate to string")
		}
		val, info, ok := ResolveExpressionOrPushEvaluation(&a.Value, &resolved, &info, binding, locally)
		if !ok || !resolved {
			return e, info, ok
		}
		newMap[kstr] = NewNode(val, binding)
	}
	return newMap, DefaultInfo(), true
}

func (e CreateMapExpr) String() string {
	result := "{"
	sep := " "
	for _, a := range e.Assignments {
		result += sep + a.String()
		sep = ", "
	}
	return result + " }"
}

type Assignment struct {
	Key   Expression
	Value Expression
}

func (e Assignment) String() string {
	return fmt.Sprintf("%s = %s", e.Key, e.Value)
}
