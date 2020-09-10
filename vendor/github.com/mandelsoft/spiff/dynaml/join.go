package dynaml

import (
	"strconv"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

func func_join(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 1 {
		return nil, info, false
	}

	args := make([]string, 0)
	for i, arg := range arguments {
		switch v := arg.(type) {
		case string:
			args = append(args, v)
		case int64:
			args = append(args, strconv.FormatInt(v, 10))
		case bool:
			args = append(args, strconv.FormatBool(v))
		case []yaml.Node:
			if i == 0 {
				return info.Error("first argument for join must be a string")
			}
			for _, elem := range v {
				switch e := elem.Value().(type) {
				case string:
					args = append(args, e)
				case int64:
					args = append(args, strconv.FormatInt(e, 10))
				case bool:
					args = append(args, strconv.FormatBool(e))
				default:
					return info.Error("elements of list(arg %d) to join must be simple values", i)
				}
			}
		case nil:
		default:
			return info.Error("argument %d to join must be simple value or list", i)
		}
	}

	return strings.Join(args[1:], args[0]), info, true
}
