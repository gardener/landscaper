package dynaml

import (
	"fmt"
	"strconv"
)

func init() {
	RegisterFunction("string", func_string)
	RegisterFunction("integer", func_integer)
	RegisterFunction("float", func_float)
	RegisterFunction("bool", func_bool)
}

func func_string(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(arguments) != 1 {
		if _, ok := arguments[0].(int64); ok {
			if len(arguments) != 2 {
				return info.Error("string for integers requires one or two arguments")
			}
		} else {
			return info.Error("string requires one argument")
		}
	}
	switch v := arguments[0].(type) {
	case string:
		return v, info, true
	case int64:
		base := 10
		if len(arguments) == 2 {
			if b, ok := arguments[1].(int64); !ok {
				return info.Error("base argument for string requires integer value")
			} else {
				base = int(b)
			}
			if 2 > base || base > 36 {
				return info.Error("base argument for string requires integer value >=2 and <=36 ")
			}
		}
		return strconv.FormatInt(v, base), info, true
	case float64:
		return fmt.Sprintf("%g", v), info, true
	case bool:
		if v {
			return "true", info, true
		}
		return "false", info, true
	default:
		return info.Error("cannot convert %T to string", v)
	}
}

func func_integer(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(arguments) != 1 {
		return info.Error("integer requires one argument")
	}
	switch v := arguments[0].(type) {
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return info.Error("%q is no integer value: %s", err)
		}
		return i, info, true
	case int64:
		return v, info, true
	case float64:
		return int64(v), info, true
	case bool:
		if v {
			return 1, info, true
		}
		return 0, info, true
	default:
		return info.Error("cannot convert %T to integer", v)
	}
}

func func_float(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(arguments) != 1 {
		return info.Error("float requires one argument")
	}
	switch v := arguments[0].(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return info.Error("%q is no float value: %s", err)
		}
		return f, info, true
	case int64:
		return float64(v), info, true
	case float64:
		return v, info, true
	case bool:
		if v {
			return float64(1), info, true
		}
		return float64(0), info, true
	default:
		return info.Error("cannot convert %T to float", v)
	}
}

func func_bool(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(arguments) != 1 {
		return info.Error("bool requires one argument")
	}
	if arguments[0] == nil {
		return false, info, true
	}
	switch v := arguments[0].(type) {
	case string:
		if v == "" {
			return false, info, true
		}
		i, err := strconv.ParseBool(v)
		if err != nil {
			return info.Error("%q is no bool value: %s", err)
		}
		return i, info, true
	case int64:
		return v != 0, info, true
	case float64:
		return v != 0, info, true
	case bool:
		return v, info, true
	default:
		return info.Error("cannot convert %T to bool", v)
	}
}
