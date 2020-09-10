package dynaml

import (
	"os"
)

func func_tempfile(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error

	info := DefaultInfo()

	if !binding.GetState().FileAccessAllowed() {
		return info.DenyOSOperation("tempfile")
	}

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("temp_file requires exactly one or two arguments")
	}

	permissions := int64(0644)
	binary := false
	if len(arguments) == 2 {
		switch v := arguments[1].(type) {
		case string:
			permissions, binary, err = getWriteOptions(v, permissions)
		case int64:
			permissions = v
		default:
			return info.Error("permissions must be given as int or int string")
		}
	}

	_, _, data, _ := getData("", binary, 0, arguments[0], true)

	name, err := binding.GetTempName(data)
	if err != nil {
		return info.Error("cannot create temporary file: %s", err)
	}

	err = binding.GetState().FileSystem().WriteFile(name, []byte(data), os.FileMode(permissions))
	if err != nil {
		return info.Error("cannot write file: %s", err)
	}

	return name, info, true
}
