package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/yaml"
)

type RangeExpr struct {
	Start Expression
	End   Expression
}

func (e RangeExpr) getRange(binding Binding, size int) (int64, int64, EvaluationInfo, bool, bool) {
	resolved := true
	info := EvaluationInfo{}

	if size < 0 && (e.Start == nil || e.End == nil) {
		info.SetError("range expression requires start and end index")
		return 0, 0, info, false, resolved
	}
	range_start := int64(0)
	range_end := int64(size - 1)
	if e.Start != nil {
		val, info, ok := ResolveIntegerExpressionOrPushEvaluation(&e.Start, &resolved, &info, binding, false)
		if !ok {
			return 0, 0, info, false, resolved
		}
		range_start = val
		if val < 0 && e.End == nil {
			range_end = -1
		}
	}
	if e.End != nil {
		val, info, ok := ResolveIntegerExpressionOrPushEvaluation(&e.End, &resolved, &info, binding, false)
		if !ok {
			return 0, 0, info, false, resolved
		}
		range_end = val
		if val < 0 && e.Start == nil {
			range_start = -int64(size)
		}
	}
	if e.Start == nil && e.End == nil {
		info.SetError("slice operator requires start or end index")
		return 0, 0, info, false, resolved
	}
	return range_start, range_end, info, true, resolved
}

func (e RangeExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	start, end, info, ok, resolved := e.getRange(binding, -1)

	if !ok {
		return nil, info, false
	}
	if !resolved {
		return e, info, true
	}

	nodes := []yaml.Node{}
	delta := int64(1)
	if start > end {
		delta = -1
	}
	for i := start; i*delta <= end*delta; i += delta {
		nodes = append(nodes, NewNode(i, binding))
	}

	return nodes, info, true
}

func (e RangeExpr) String() string {
	return fmt.Sprintf("[%s..%s]", e.Start, e.End)
}
