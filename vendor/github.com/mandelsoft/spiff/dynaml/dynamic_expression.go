package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type DynamicExpr struct {
	Root  Expression
	Index Expression
}

func (e DynamicExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {

	// if root is a reference expression and the type is known allow for element selection if element is resolved
	// regardless of the resolution state of the root
	// enables .["dyn"]
	_, isRef := e.Root.(ReferenceExpr)

	root, info, ok := e.Root.Evaluate(binding, locally || isRef)
	if !ok {
		return nil, info, false
	}

	if !isLocallyResolvedValue(root, binding) {
		return e, info, true
	}

	locally = locally || info.Raw
	/*
		if !locally && !isResolvedValue(root, binding) {
			info.Issue = yaml.NewIssue("'%s' unresolved", e.Root)
			return e, info, true
		}
	*/

	dyn, infoe, ok := e.Index.Evaluate(binding, locally)

	info.Join(infoe)
	if !ok {
		return nil, info, false
	}
	if !isResolvedValue(dyn, binding) {
		return e, info, true
	}

	debug.Debug("dynamic reference: %+v\n", dyn)

	if a, ok := dyn.([]yaml.Node); ok {
		if len(a) == 1 {
			dyn = a[0].Value()
		}
	}
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
		if len(v) == 0 {
			return info.Error("at least one index or field name required for reference qualifier")
		}
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

	t, info, ok := NewReferenceExpr(qual...).find(func(end int, path []string) (yaml.Node, bool) {
		return yaml.Find(NewNode(root, nil), binding.GetFeatures(), path[:end+1]...)
	}, binding, true)

	if !ok {
		return nil, info, false
	}
	if isResolvedValue(t, binding) {
		return t, info, true
	}
	return e, info, true
}

func (e DynamicExpr) String() string {
	return fmt.Sprintf("%s.%s", e.Root, e.Index)
}
