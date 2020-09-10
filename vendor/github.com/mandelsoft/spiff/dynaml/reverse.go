package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	RegisterFunction("reverse", func_reverse)
}

func func_reverse(arguments []interface{}, binding Binding) (result interface{}, info EvaluationInfo, ok bool) {
	info = DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("reverse takes one list argument")
	}

	list, ok := arguments[0].([]yaml.Node)
	if !ok {
		return info.Error("argument for reverse must be a list")
	}

	max := len(list) - 1

	for i := 0; i < (max+1)/2; i++ {
		list[i], list[max-i] = list[max-i], list[i]
	}
	return list, info, true
}
