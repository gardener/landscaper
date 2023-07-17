package semver

import (
	"github.com/Masterminds/semver/v3"

	. "github.com/mandelsoft/spiff/dynaml"
)

const F_IncMajor = "semverincmajor"
const F_IncMinor = "semverincminor"
const F_IncPatch = "semverincpatch"

func init() {
	RegisterFunction(F_IncMajor, func_incmajor)
	RegisterFunction(F_IncMinor, func_incminor)
	RegisterFunction(F_IncPatch, func_incpatch)
}

func func_incmajor(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver(F_IncMajor, func(v *semver.Version) interface{} { r := v.IncMajor(); return r.Original() }, arguments, binding)
}

func func_incminor(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver(F_IncMinor, func(v *semver.Version) interface{} { r := v.IncMinor(); return r.Original() }, arguments, binding)
}

func func_incpatch(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver(F_IncPatch, func(v *semver.Version) interface{} { r := v.IncPatch(); return r.Original() }, arguments, binding)
}
