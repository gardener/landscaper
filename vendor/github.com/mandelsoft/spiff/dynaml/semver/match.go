package semver

import (
	. "github.com/mandelsoft/spiff/dynaml"
)

const F_Match = "semvermatch"

func init() {
	RegisterFunction(F_Match, func_match)
}

func func_match(args []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(args) == 0 {
		return info.Error("%s requires at least one arguments", F_Match)
	}

	ok, _, err, _ := validate(F_Match, false, args[0], binding, args[1:]...)
	if err != nil {
		return info.Error("%s", err)
	}
	return ok, info, true
}
