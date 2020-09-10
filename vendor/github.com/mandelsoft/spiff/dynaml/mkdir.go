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

	if !binding.GetState().FileAccessAllowed() {
		return info.DenyOSOperation("mkdir")
	}

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("mkdir requires one or two arguments")
	}
	path, _, ok := getArg(0, arguments[0], false)
	if !ok {
		comps, ok := arguments[0].([]yaml.Node)
		if !ok {
			return info.Error("path argument must be a non-empty string")
		}
		for i, a := range comps {
			comp, _, ok := getArg(i, a.Value(), false)
			if !ok || comp == "" {
				return info.Error("path component %d must be a non-empty string", i)
			}
			path = filepath.Join(path, comp)
		}
	}
	if path == "" {
		return info.Error("path argument must not be empty")
	}

	permissions := int64(0755)
	binary := false
	if len(arguments) == 2 {
		switch v := arguments[1].(type) {
		case string:
			permissions, binary, err = getWriteOptions(v, permissions)
			if err != nil {
				return info.Error("%s", err)
			}
			if binary {
				return info.Error("binary option not supported for mkdir")
			}
		case int64:
			permissions = v
		default:
			return info.Error("permissions must be given as int or int string")
		}
	}

	err = binding.GetState().FileSystem().MkdirAll(path, os.FileMode(permissions))
	if err != nil {
		return info.Error("cannot create directory %q: %s", path, err)
	}

	return path, info, true
}
