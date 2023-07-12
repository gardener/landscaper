package semver

import (
	"github.com/Masterminds/semver/v3"

	. "github.com/mandelsoft/spiff/dynaml"
)

const F_Major = "semvermajor"
const F_Minor = "semverminor"
const F_Patch = "semverpatch"
const F_Prerelease = "semverprerelease"
const F_Metadata = "semvermetadata"
const F_Release = "semverrelease"
const F_Normalize = "semver"

func init() {
	RegisterFunction(F_Major, func_major)
	RegisterFunction(F_Minor, func_minor)
	RegisterFunction(F_Patch, func_patch)
	RegisterFunction(F_Prerelease, func_prerelease)
	RegisterFunction(F_Metadata, func_metadata)
	RegisterFunction(F_Release, func_release)
	RegisterFunction(F_Normalize, func_normalize)
}

func parse(name string, arg interface{}) (*semver.Version, EvaluationInfo) {
	info := DefaultInfo()
	s, ok := arg.(string)
	if !ok {
		info.SetError("%s requires one string argument, but got %s", name, ExpressionType(arg))
		return nil, info
	}
	v, err := semver.NewVersion(s)
	if err != nil {
		info.SetError("%s: %q: %s", name, s, err)
		return nil, info
	}
	return v, info
}

func func_semver(name string, get func(version *semver.Version) interface{}, args []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver2(name, get, nil, args, binding)
}

func func_semver2(name string, get func(version *semver.Version) interface{}, set func(version *semver.Version, arg interface{}) (interface{}, EvaluationInfo, bool), args []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if set != nil && len(args) > 1 {
		if len(args) > 2 {
			return info.Error("%s requires one semver argument and an optional value", name)
		}
	} else {
		if len(args) != 1 {
			return info.Error("%s requires one semver argument", name)
		}
	}
	v, info := parse(name, args[0])

	if v == nil {
		return nil, info, false
	}
	if set != nil && len(args) == 2 {
		return set(v, args[1])
	}
	return get(v), info, true
}

func setter(name, attr string, f func(v *semver.Version, s string) (semver.Version, error)) func(v *semver.Version, val interface{}) (interface{}, EvaluationInfo, bool) {
	return func(v *semver.Version, val interface{}) (interface{}, EvaluationInfo, bool) {
		info := DefaultInfo()
		s, err := StringValue(attr, val)
		if err != nil {
			return info.Error("%s: %s", name, err)
		}
		r, err := f(v, s)
		if err != nil {
			return info.Error("%s: invalid %s %q: %s", name, attr, s, err)
		}
		return r.Original(), info, true
	}
}

////////////////////////////////////////////////////////////////////////////////

func func_major(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver(F_Major, func(v *semver.Version) interface{} { return int64(v.Major()) }, arguments, binding)
}

func func_minor(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver(F_Minor, func(v *semver.Version) interface{} { return int64(v.Minor()) }, arguments, binding)
}

func func_patch(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver(F_Patch, func(v *semver.Version) interface{} { return int64(v.Patch()) }, arguments, binding)
}

func func_prerelease(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver2(F_Prerelease,
		func(v *semver.Version) interface{} {
			return v.Prerelease()
		},
		setter(F_Prerelease, "prerelease", func(v *semver.Version, s string) (semver.Version, error) { return v.SetPrerelease(s) }),
		arguments, binding)
}

func func_metadata(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver2(F_Metadata,
		func(v *semver.Version) interface{} {
			return v.Metadata()
		},
		setter(F_Metadata, "metadata", func(v *semver.Version, s string) (semver.Version, error) { return v.SetMetadata(s) }),
		arguments, binding)
}

func func_release(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver2(F_Release,
		func(v *semver.Version) interface{} {
			r, _ := v.SetMetadata("")
			r, _ = r.SetPrerelease("")
			return r.Original()
		},
		setter(F_Release, "release",
			func(v *semver.Version, s string) (semver.Version, error) {
				n, err := semver.NewVersion(s)
				if err != nil {
					return semver.Version{}, err
				}
				r, _ := n.SetMetadata(v.Metadata())
				r, _ = r.SetPrerelease(v.Prerelease())
				return r, nil
			}),
		arguments, binding)
}

func func_normalize(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_semver(F_Prerelease, func(v *semver.Version) interface{} { return v.String() }, arguments, binding)
}
