package dynaml

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mandelsoft/spiff/legacy/candiedyaml"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

func func_exec(cached bool, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 1 {
		return info.Error("exec: argument required")
	}
	if !binding.GetState().OSAccessAllowed() {
		return info.DenyOSOperation("exec")
	}
	args := []string{}
	debug.Debug("exec: found %d arguments for call\n", len(arguments))
	for i, arg := range arguments {
		list, ok := arg.([]yaml.Node)
		if i == 0 && ok {
			debug.Debug("exec: found array as first argument\n")
			if len(arguments) == 1 && len(list) > 0 {
				// handle single list argument to gain command and argument
				for j, arg := range list {
					v, _, ok := getArg(j, arg.Value(), j != 0)
					if !ok {
						return info.Error("command argument must be string")
					}
					args = append(args, v)
				}
			} else {
				return info.Error("list not allowed for command argument")
			}
		} else {
			v, _, ok := getArg(i, arg, i != 0)
			if !ok {
				return info.Error("command argument must be string")
			}
			args = append(args, v)
		}
	}
	result, err := cachedExecute(cached, nil, args)
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

func getArg(key interface{}, value interface{}, yaml bool) (string, bool, bool) {
	debug.Debug("arg %v: %+v\n", key, value)
	switch v := value.(type) {
	case string:
		return v, true, true
	case int64:
		return strconv.FormatInt(v, 10), false, true
	case bool:
		return strconv.FormatBool(v), false, true
	default:
		if !yaml || value == nil {
			return "", false, false
		}
		yaml, err := candiedyaml.Marshal(NewNode(value, nil))
		if err != nil {
			log.Fatalln("error marshalling manifest:", err)
		}
		return "---\n" + string(yaml), false, true
	}
}

var cache = make(map[string][]byte)

type Bytes interface {
	Bytes() []byte
}

func cachedExecute(cached bool, content *string, args []string) ([]byte, error) {
	h := md5.New()
	if content != nil {
		h.Write([]byte(*content))
	}
	for _, arg := range args {
		h.Write([]byte(arg))
	}
	hash := fmt.Sprintf("%x", h.Sum(nil))
	if cached {
		result := cache[hash]
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
	cache[hash] = result
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
