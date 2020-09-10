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
	file, _, ok := getArg(0, arguments[0], false)
	if !ok || file == "" {
		return info.Error("file argument must be a non-empty string")
	}

	permissions := int64(0644)
	binary := false
	if len(arguments) == 3 {
		switch v := arguments[2].(type) {
		case string:
			permissions, binary, err = getWriteOptions(v, 0644)
			if err != nil {
				return info.Error("%s", err)
			}
		case int64:
			permissions = v
		default:
			return info.Error("permissions must be given as int or int string")
		}
	}

	file, raw, data, _ := getData(file, binary, 1, arguments[1], true)

	err = binding.GetState().FileSystem().WriteFile(file, data, os.FileMode(permissions))
	if err != nil {
		return info.Error("cannot write file: %s", err)
	}

	return raw, info, true
}

func getWriteOptions(v string, def int64) (permissions int64, binary bool, err error) {
	permissions = def
	opts := strings.Split(v, ",")
	for i, o := range opts {
		o = strings.TrimSpace(o)
		if o == "binary" || o == "#" {
			binary = true
			opts = append(opts[:i], opts[i+1:]...)
		}
	}
	if len(opts) > 1 {
		err = fmt.Errorf("invalid options %v, expecting file mode", opts)
		return
	}
	if len(opts) == 1 {
		v = opts[0]

		base := 10
		if strings.HasPrefix(v, "0") {
			base = 8
		}
		permissions, err = strconv.ParseInt(v, base, 64)
		if err != nil {
			err = fmt.Errorf("permissions must be given as int or int string: %s", err)
		}
	}
	return
}

func getData(file string, binary bool, key interface{}, value interface{}, yaml bool) (string, string, []byte, bool) {
	var data []byte
	var err error

	str, isstr, ok := getArg(key, value, true)
	if binary || (isstr && hasTag(file, "#")) {
		data, err = base64.StdEncoding.DecodeString(str)
		if err != nil {
			data = []byte(str)
		}
	} else {
		data = []byte(str)
	}
	return removeTags(file), str, data, ok
}
