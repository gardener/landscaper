package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type DynamicExpr struct {
	Expression Expression
	Reference  Expression
}

func (e DynamicExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {

	root, info, ok := e.Expression.Evaluate(binding, locally)
	if !ok {
		return nil, info, false
	}

	locally = locally || info.Raw

	if !isLocallyResolvedValue(root) {
		return e, info, true
	}

	if !locally && !isResolvedValue(root) {
		return e, info, true
	}

	dyn, infoe, ok := e.Reference.Evaluate(binding, locally)
	info.Join(infoe)
	if !ok {
		return nil, info, false
	}
	if !isResolvedValue(dyn) {
		return e, info, true
	}

	debug.Debug("dynamic reference: %+v\n", dyn)

	var qual []string
	switch v := dyn.(type) {
	case int64:
		_, ok := root.([]yaml.Node)
		if !ok {
			return info.Error("index requires array expression")
		}
		qual = []string{fmt.Sprintf("[%d]", v)}
	case string:
		qual = []string{v}
	case []yaml.Node:
		qual = make([]string, len(v))
		for i, e := range v {
			switch v := e.Value().(type) {
			case int64:
				qual[i] = fmt.Sprintf("[%d]", v)
			case string:
				qual[i] = v
			default:
				return info.Error("index or field name required for reference qualifier")
			}
		}
	default:
		return info.Error("index or field name required for reference qualifier")
	}
	return ReferenceExpr{qual}.find(func(end int, path []string) (yaml.Node, bool) {
		return yaml.Find(NewNode(root, nil), path[:end+1]...)
	}, binding, locally)
}

func (e DynamicExpr) String() string {
	return fmt.Sprintf("%s.[%s]", e.Expression, e.Reference)
}
