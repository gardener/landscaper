package dynaml

import (
	"fmt"

	//"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type SliceExpr struct {
	Expression Expression
	Range      RangeExpr
}

func (e SliceExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {

	root, info, ok := e.Expression.Evaluate(binding, locally)
	if !ok {
		return nil, info, false
	}
	if !isLocallyResolvedValue(root, binding) {
		return e, info, true
	}
	if !locally && !isResolvedValue(root, binding) {
		return e, info, true
	}

	array, ok := root.([]yaml.Node)
	if !ok {
		return info.Error("slice requires array expression")
	}

	start, end, infoe, ok, resolved := e.Range.getRange(binding, len(array))
	info.Join(infoe)
	if !ok {
		return nil, info, ok
	}
	if !resolved {
		return e, info, ok
	}
	if start > end {
		return []yaml.Node{}, info, ok
	}

	if start < 0 {
		if end >= 0 {
			return info.Error("mixed negative and non-negative range not possible for slice")
		}
		if start < -int64(len(array)) {
			return info.Error("slice out of range (%d < -length %d)", start, len(array))
		}
		result := make([]yaml.Node, end-start+1)
		for i := start; i <= end; i++ {
			result[i-start] = array[i+int64(len(array))]
		}
		return result, info, true
	} else {
		if end >= int64(len(array)) {
			return info.Error("slice out of range (%d >= length %d)", end, len(array))
		}
		result := make([]yaml.Node, end-start+1)
		for i := start; i <= end; i++ {
			result[i-start] = array[i]
		}
		return result, info, true
	}
}

func (e SliceExpr) String() string {
	return fmt.Sprintf("%s.[%s]", e.Expression, e.Range)
}
