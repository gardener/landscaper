package dynaml

import (
	"net/url"
	"path"
)

func init() {
	RegisterFunction("basename", func_basename)
	RegisterFunction("dirname", func_dirname)
}

func func_basename(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("function basename takes exactly one argument")
	}

	switch val := arguments[0].(type) {
	case string:
		u, err := url.Parse(val)
		if err == nil {
			if u.Path == "" {
				return "/", info, true
			}
			return path.Base(u.Path), info, true
		}
		return path.Base(val), info, true
	default:
		return info.Error("string argument expected for function basename")
	}
}

func func_dirname(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("function dirname takes exactly one argument")
	}

	switch val := arguments[0].(type) {
	case string:
		u, err := url.Parse(val)
		if err == nil {
			if u.Path == "" {
				return "/", info, true
			}
			return path.Dir(u.Path), info, true
		}
		return path.Dir(val), info, true
	default:
		return info.Error("string argument expected for function dirname")
	}
}
