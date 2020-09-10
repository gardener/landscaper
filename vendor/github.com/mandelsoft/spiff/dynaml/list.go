package dynaml

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

type ListExpr struct {
	Contents []Expression
}

func (e ListExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true

	values, info, ok := ResolveExpressionListOrPushEvaluation(&e.Contents, &resolved, nil, binding, false)

	if !ok {
		return nil, info, false
	}
	if !resolved {
		return e, info, true
	}

	nodes := []yaml.Node{}
	for i, _ := range values {
		nodes = append(nodes, NewNode(values[i], binding))
	}
	return nodes, info, true
}

func (e ListExpr) String() string {
	vals := make([]string, len(e.Contents))
	for i, e := range e.Contents {
		vals[i] = fmt.Sprintf("%s", e)
	}

	return fmt.Sprintf("[%s]", strings.Join(vals, ", "))
}
