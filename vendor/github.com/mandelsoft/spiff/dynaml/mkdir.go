package dynaml

import (
	"os"
	"path/filepath"

	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	RegisterFunction("mkdir", func_mkdir)
}

func func_mkdir(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error
	info := DefaultInfo()

	wopt := WriteOpts{
		Permissions: 0755,
	}

	if !binding.GetState().FileAccessAllowed() {
		return info.DenyOSOperation("mkdir")
	}

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("mkdir requires one or two arguments")
	}
	path, _, err := getArg(0, arguments[0], wopt, false)
	if err != nil {
		comps, ok := arguments[0].([]yaml.Node)
		if !ok {
			return info.Error("path argument must be a non-empty string")
		}
		for i, a := range comps {
			comp, _, err := getArg(i, a.Value(), wopt, false)
			if err != nil || comp == "" {
				return info.Error("path component %d must be a non-empty string", i)
			}
			path = filepath.Join(path, comp)
		}
	}
	if path == "" {
		return info.Error("path argument must not be empty")
	}

	if len(arguments) == 2 {
		wopt, err = getWriteOptions(arguments[1], wopt, true)
	}
	if err == nil {
		err = binding.GetState().FileSystem().MkdirAll(path, os.FileMode(wopt.Permissions))
		if err == nil {
			return path, info, true
		}
	}
	return info.Error("cannot create directory %q: %s", path, err)
}
