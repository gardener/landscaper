package dynaml

import ()

func func_substr(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) > 3 || len(arguments) < 2 {
		return info.Error("substr takes two to three arguments")
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for substr must be a string")
	}
	start, ok := arguments[1].(int64)
	if !ok {
		return info.Error("second argument for substr must be an integer")
	}
	if start < 0 {
		start = int64(len(str)) + start
	}
	var end int64 = int64(len(str))
	if len(arguments) >= 3 {
		end, ok = arguments[2].(int64)
		if !ok {
			return info.Error("third argument for substr must be an integer")
		}
		if end < 0 {
			end = int64(len(str)) + end
		}
	}

	if int64(len(str)) < end {
		return info.Error("substr effective end index (%d) exceeds string length (%d)", end, len(str))
	}
	if start < 0 {
		return info.Error("negative substr effective start index (%d)", start)
	}
	if start > end {
		return info.Error("substr start index (%d) aftsre end index (%d) ", start, end)
	}

	return str[start:end], info, true
}
