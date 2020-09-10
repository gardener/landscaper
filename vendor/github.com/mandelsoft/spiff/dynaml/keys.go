package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

func func_keys(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("one argument required for keys")
	}

	m, ok := arguments[0].(map[string]yaml.Node)
	if !ok {
		return info.Error("map argument required for keys")
	}

	result := []yaml.Node{}
	for _, k := range getSortedKeys(m) {
		result = append(result, NewNode(k, binding))
	}
	return result, info, true
}
