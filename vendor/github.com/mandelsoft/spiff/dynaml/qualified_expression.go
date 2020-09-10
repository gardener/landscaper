package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type QualifiedExpr struct {
	Expression Expression
	Reference  ReferenceExpr
}

func (e QualifiedExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {

	root, info, ok := e.Expression.Evaluate(binding, locally)
	if !ok {
		debug.Debug("base of qualified expression failed: %s\n", info.Issue.Issue)
		return nil, info, false
	}
	locally = locally || info.Raw
	if !isLocallyResolvedValue(root) {
		debug.Debug("not locally resolved: %v\n", root)
		return e, info, true
	}
	if !locally && !isResolvedValue(root) {
		debug.Debug("not resoved: %v\n", root)
		return e, info, true
	}

	debug.Debug("qualified reference (%t): %v\n", locally, e.Reference.Path)
	return e.Reference.find(func(end int, path []string) (yaml.Node, bool) {
		return yaml.Find(NewNode(root, nil), e.Reference.Path[0:end+1]...)
	}, binding, locally)
}

func (e QualifiedExpr) String() string {
	return fmt.Sprintf("(%s).%s", e.Expression, e.Reference)
}
