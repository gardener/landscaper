package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type ProjectionExpr struct {
	Expression Expression
	Value      *ProjectionValue
	Projection Expression
}

func (e ProjectionExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true
	value, infoa, ok := ResolveExpressionOrPushEvaluation(&e.Expression, &resolved, nil, binding, false)
	if !ok {
		return nil, infoa, false
	}
	if !resolved {
		return e, infoa, false
	}
	switch v := value.(type) {
	case []yaml.Node:
		if _, ok := e.Projection.(ProjectionValueExpr); ok {
			return v, infoa, true
		} else {
			newList := make([]yaml.Node, len(v))
			for index, entry := range v {
				result, _, ok := projectValue(e.Value, entry, e.Projection, binding, locally)
				if !ok {
					return nil, infoa, false
				}
				if !isLocallyResolvedValue(newList[index]) {
					return e, infoa, true
				}
				if !locally && !isResolvedValue(newList[index]) {
					return e, infoa, true
				}
				newList[index] = NewNode(result, binding)
			}
			return newList, infoa, true
		}
	case map[string]yaml.Node:
		newList := make([]yaml.Node, len(v))
		index := 0
		for _, key := range getSortedKeys(v) {
			result, _, ok := projectValue(e.Value, v[key], e.Projection, binding, locally)
			if !ok {
				return nil, infoa, false
			}
			if !isLocallyResolvedValue(newList[index]) {
				return e, infoa, true
			}
			if !locally && !isResolvedValue(newList[index]) {
				return e, infoa, true
			}
			newList[index] = NewNode(result, binding)
			index++
		}
		return newList, infoa, true
	default:
		return infoa.Error("only map or list allowed for projection")
	}
}

func projectValue(ref *ProjectionValue, value yaml.Node, expr Expression, binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	ref.Value = value.Value()
	root, info, ok := expr.Evaluate(binding, locally)
	if !ok {
		return nil, info, false
	}
	return root, info, true
}

func (e ProjectionExpr) String() string {
	return fmt.Sprintf("%s.[*] %s", e.Expression, e.Projection)
}

type ProjectionValue struct {
	Value interface{}
}

type ProjectionValueExpr struct {
	Value *ProjectionValue
}

func (e ProjectionValueExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	debug.Debug("projection of value: %+v\n", e.Value.Value)
	return e.Value.Value, DefaultInfo(), true
}

func (e ProjectionValueExpr) String() string {
	return ""
}
