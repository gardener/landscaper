package dynaml

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func func_write(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error
	info := DefaultInfo()

	if len(arguments) < 2 || len(arguments) > 3 {
		return info.Error("write requires two or three arguments")
	}
	if !binding.GetState().FileAccessAllowed() {
		return info.DenyOSOperation("write")
	}

	wopt := WriteOpts{
		Permissions: 0644,
	}

	file, _, err := getArg(0, arguments[0], wopt, false)
	if err != nil || file == "" {
		return info.Error("file argument must be a non-empty string")
	}

	if len(arguments) == 3 {
		wopt, err = getWriteOptions(arguments[2], wopt, false)
	}
	if err == nil {
		file, raw, data, _ := getData(file, wopt, 1, arguments[1], true)

		err = binding.GetState().FileSystem().WriteFile(file, data, os.FileMode(wopt.Permissions))
		if err == nil {
			return raw, info, true
		}
	}
	return info.Error("cannot write file: %s", err)
}

type WriteOpts struct {
	Binary      bool
	Multi       bool
	Permissions int64
}

func getWriteOptions(arg interface{}, wopt WriteOpts, permonly bool) (WriteOpts, error) {
	var err error
	switch v := arg.(type) {
	case string:
		opts := strings.Split(v, ",")
		for i := 0; i < len(opts); i++ {
			o := strings.TrimSpace(opts[i])
			switch o {
			case "binary", "#":
				wopt.Binary = true
				opts = append(opts[:i], opts[i+1:]...)
				i--
			case "multiyaml":
				wopt.Multi = true
				opts = append(opts[:i], opts[i+1:]...)
				i--
			}
		}
		if len(opts) > 1 {
			return wopt, fmt.Errorf("invalid options %v, expecting file mode", opts)
		}
		if len(opts) == 1 {
			v = opts[0]

			base := 10
			if strings.HasPrefix(v, "0") {
				base = 8
			}
			wopt.Permissions, err = strconv.ParseInt(v, base, 64)
			if err != nil {
				err = fmt.Errorf("permissions must be given as int or int string: %s", err)
			}
		}
	case int64:
		wopt.Permissions = v
	default:
		err = fmt.Errorf("permissions must be given as int or int string, found %s", ExpressionType(arg))
	}
	if permonly {
		if wopt.Binary {
			return wopt, fmt.Errorf("binary option not support -> only permissions")
		}
		if wopt.Multi {
			return wopt, fmt.Errorf("multi option not support -> only permissions")
		}
	}
	return wopt, err
}

func FilePath(file string) string {
	if strings.HasPrefix(file, "~/") {
		home := os.Getenv("HOME")
		if home != "" {
			file = home + file[1:]
		}
	}
	return file
}

func getData(file string, wopt WriteOpts, key interface{}, value interface{}, yaml bool) (string, string, []byte, bool) {
	var data []byte
	var err error

	str, isstr, err := getArg(key, value, wopt, true)
	if wopt.Binary || (isstr && hasTag(file, "#")) {
		data, err = base64.StdEncoding.DecodeString(str)
		if err != nil {
			data = []byte(str)
		}
	} else {
		data = []byte(str)
	}
	return FilePath(removeTags(file)), str, data, err == nil
}
