package dynaml

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mandelsoft/spiff/legacy/candiedyaml"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

func func_exec(cached bool, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var cache ExecCache

	info := DefaultInfo()

	if len(arguments) < 1 {
		return info.Error("exec: argument required")
	}
	if !binding.GetState().OSAccessAllowed() {
		return info.DenyOSOperation("exec")
	}
	if cached {
		cache = binding.GetState().GetExecCache()
	}
	args := []string{}
	wopt := WriteOpts{}
	debug.Debug("exec: found %d arguments for call\n", len(arguments))
	for i, arg := range arguments {
		list, ok := arg.([]yaml.Node)
		if i == 0 && ok {
			debug.Debug("exec: found array as first argument\n")
			if len(arguments) == 1 && len(list) > 0 {
				// handle single list argument to gain command and argument
				for j, arg := range list {
					v, _, err := getArg(j, arg.Value(), wopt, j != 0)
					if err != nil {
						return info.Error("invalid command argument: %s", err)
					}
					args = append(args, v)
				}
			} else {
				return info.Error("list not allowed for command argument")
			}
		} else {
			v, _, err := getArg(i, arg, wopt, i != 0)
			if err != nil {
				return info.Error("invalid command argument: %s", err)
			}
			args = append(args, v)
		}
	}
	result, err := cachedExecute(cache, nil, args)
	if err != nil {
		return info.Error("execution '%s' failed", args[0])
	}

	return convertOutput(result)
}

func convertOutput(data []byte) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	str := string(data)
	debug.Debug("DATA--------------------------\n")
	debug.Debug("%s\n", str)
	debug.Debug("------------------------------\n")
	execYML, err := yaml.Parse("exec", data)
	if execYML != nil && err == nil && (isMap(execYML) || isMap(execYML) || strings.HasPrefix(str, "---\n")) {
		debug.Debug("exec: found yaml result %+v\n", execYML)
		return execYML.Value(), info, true
	} else {
		for strings.HasSuffix(str, "\n") {
			str = str[:len(str)-1]
		}
		int64YML, err := strconv.ParseInt(str, 10, 64)
		if err == nil {
			debug.Debug("exec: found integer result: %d\n", int64YML)
			return int64YML, info, true
		}
		debug.Debug("exec: found string result: %s\n", string(data))
		return str, info, true
	}
}

func getArg(key interface{}, value interface{}, wopt WriteOpts, allowyaml bool) (string, bool, error) {
	debug.Debug("arg %v: %+v\n", key, value)
	switch v := value.(type) {
	case string:
		return v, true, nil
	case int64:
		return strconv.FormatInt(v, 10), false, nil
	case float64:
		return strconv.FormatFloat(v, 'e', 64, 64), false, nil
	case bool:
		return strconv.FormatBool(v), false, nil
	default:
		if !allowyaml || value == nil {
			return "", false, fmt.Errorf("yaml or empty data not supported")
		}
		if wopt.Multi {
			if list, ok := value.([]yaml.Node); ok {
				result := ""
				for i, d := range list {
					yaml, err := candiedyaml.Marshal(d)
					if err != nil {
						return "", false, fmt.Errorf("error marshalling entry %d: %s", i, err)
					}
					result = result + "---\n" + string(yaml)
				}
				return result, false, nil
			} else {
				return "", false, fmt.Errorf("multi document mode requires a list")
			}
		}
		yaml, err := candiedyaml.Marshal(NewNode(value, nil))
		if err != nil {
			return "", false, fmt.Errorf("error marshalling manifest: %s", err)
		}
		return "---\n" + string(yaml), false, nil
	}
}

type Bytes interface {
	Bytes() []byte
}

func cachedExecute(cache ExecCache, content *string, args []string) ([]byte, error) {
	h := md5.New()
	if content != nil {
		h.Write([]byte(*content))
	}
	args[0] = FilePath(args[0])
	for _, arg := range args {
		h.Write([]byte(arg))
	}
	hash := fmt.Sprintf("%x", h.Sum(nil))
	if cache != nil {
		cache.Lock()
		defer cache.Unlock()
		result := cache.Get(hash)
		if result != nil {
			debug.Debug("exec: reusing cache %s for %v\n", hash, args)
			return result, nil
		}
	}
	debug.Debug("exec: calling %v\n", args)
	cmd := exec.Command(args[0], args[1:]...)
	if content != nil {
		cmd.Stdin = bytes.NewReader([]byte(*content))
	}
	result, err := cmd.Output()
	stderr := string(cmd.Stderr.(Bytes).Bytes())
	if stderr != "" {
		fmt.Fprintf(os.Stderr, "exec: calling %v\n", args)
		fmt.Fprintf(os.Stderr, "  error: %v\n", stderr)
	}
	if cache != nil {
		cache.Set(hash, result)
	}
	return result, err
}

func isMap(n yaml.Node) bool {
	if n == nil || n.Value() == nil {
		return false
	}
	_, ok := n.Value().(map[string]yaml.Node)
	return ok
}

func isList(n yaml.Node) bool {
	if n == nil || n.Value() == nil {
		return false
	}
	_, ok := n.Value().([]yaml.Node)
	return ok
}
