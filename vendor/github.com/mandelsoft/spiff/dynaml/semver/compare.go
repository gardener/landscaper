package semver

import (
	. "github.com/mandelsoft/spiff/dynaml"
)

const F_Compare = "semvercmp"

func init() {
	RegisterFunction(F_Compare, func_compare)
}

func func_compare(args []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(args) != 2 {
		return info.Error("%s requires two arguments", F_Compare)
	}
	v1, info := parse(F_Compare, args[0])
	if v1 == nil {
		return nil, info, false
	}
	v2, info := parse(F_Compare, args[1])
	if v2 == nil {
		return nil, info, false
	}
	return v1.Compare(v2), info, true
}
