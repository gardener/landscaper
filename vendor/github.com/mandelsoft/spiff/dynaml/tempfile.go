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

	wopt := WriteOpts{
		Permissions: 0644,
	}
	if len(arguments) == 2 {
		wopt, err = getWriteOptions(arguments[1], wopt, false)
	}
	if err != nil {
		return info.Error("cannot create temporary file: %s", err)
	}
	_, _, data, _ := getData("", wopt, 0, arguments[0], true)

	name, err := binding.GetTempName(data)
	if err != nil {
		return info.Error("cannot create temporary file: %s", err)
	}

	err = binding.GetState().FileSystem().WriteFile(name, []byte(data), os.FileMode(wopt.Permissions))
	if err != nil {
		return info.Error("cannot write file: %s", err)
	}

	return name, info, true
}
