package dynaml

import (
	"os"
	"strconv"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

var environ []string = os.Environ()

func ReloadEnv() {
	environ = os.Environ()
}

func getenv(name string) (string, bool) {
	name += "="
	for _, s := range environ {
		if strings.HasPrefix(s, name) {
			return s[len(name):], true
		}
	}
	return "", false
}

func func_env(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
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
			return info.Error("env argument %d must be simple value or list", i)
		}
	}

	if len(args) == 1 {
		s, ok := getenv(args[0])
		if ok {
			return s, info, ok
		} else {
			return info.Error("environment variable '%s' not set", args[0])
		}
	} else {
		m := make(map[string]yaml.Node)
		for _, n := range args {
			s, ok := getenv(n)
			if ok {
				m[n] = NewNode(s, nil)
			}
		}
		return m, info, true
	}
}
