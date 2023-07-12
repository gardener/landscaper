package dynaml

import (
	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

func func_pipe(cached bool, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var cache ExecCache
	info := DefaultInfo()

	if len(arguments) <= 2 {
		return info.Error("pipe requires at least two arguments")
	}
	if !binding.GetState().OSAccessAllowed() {
		return info.DenyOSOperation("pipe")
	}
	if cached {
		cache = binding.GetState().GetExecCache()
	}
	args := []string{}
	wopt := WriteOpts{}
	debug.Debug("pipe: found %d arguments for call\n", len(arguments))
	for i, arg := range arguments {
		if i == 1 {
			list, ok := arg.([]yaml.Node)
			if ok {
				debug.Debug("exec: found array as second argument\n")
				if len(arguments) == 2 && len(list) > 0 {
					// handle single list argument to gain command and argument
					for j, arg := range list {
						v, _, err := getArg(j, arg.Value(), wopt, j != 0)
						if err != nil {
							return info.Error("command argument must be string")
						}
						args = append(args, v)
					}
				} else {
					return info.Error("list not allowed for command argument")
				}
			} else {
				v, _, err := getArg(i, arg, wopt, i != 1)
				if err != nil {
					return info.Error("command argument must be string")
				}
				args = append(args, v)
			}
		} else {
			v, _, err := getArg(i, arg, wopt, i != 1)
			if err != nil {
				return info.Error("command argument must be string")
			}
			args = append(args, v)
		}
	}
	result, err := cachedExecute(cache, &args[0], args[1:])
	if err != nil {
		return info.Error("execution '%s' failed", args[1])
	}

	return convertOutput(result)
}
