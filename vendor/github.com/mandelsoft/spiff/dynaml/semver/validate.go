package semver

import (
	"github.com/Masterminds/semver/v3"

	. "github.com/mandelsoft/spiff/dynaml"
)

const F_Validate = "semvervalidate"
const V_Validate = "semver"

func init() {
	RegisterFunction(F_Validate, func_validate)
	RegisterValidator(V_Validate, validate_semver)
}

func func_validate(args []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(args) == 0 {
		return info.Error("%s requires at least one arguments", F_Validate)
	}

	ok, msg, err, _ := validate(F_Validate, false, args[0], binding, args[1:]...)

	if err != nil {
		return info.Error("%s: %q: %s", F_Validate, args[0], err)
	}
	if !ok {
		return info.Error("%s: %q: %s", F_Validate, args[0], msg)
	}
	return args[0], info, true
}

func validate_semver(value interface{}, binding Binding, args ...interface{}) (bool, string, error, bool) {
	return validate(V_Validate, true, value, binding, args...)
}

func validate(name string, noerr bool, value interface{}, binding Binding, args ...interface{}) (bool, string, error, bool) {
	v, info := parse(name, value)
	if v == nil {
		if noerr {
			return ValidatorResult(false, "%s", info.GetError())
		}
		return ValidatorErrorf("%s", info.GetError())
	}
	for i, a := range args {
		constraint, ok := a.(string)
		if !ok {
			return ValidatorErrorf("%s: constraint argument %d must be string", name, i)
		}

		c, err := semver.NewConstraint(constraint)
		if err != nil {
			return ValidatorErrorf("%s: invalid constraint %q[%d]: %s", name, constraint, i, err)
		}
		ok, msgs := c.Validate(v)
		if !ok {
			return ValidatorResult(ok, "%v", msgs)
		}
	}
	if len(args) > 0 {
		return ValidatorResult(true, "matches contraint")
	}
	return true, "is semantic version", nil, true
}
