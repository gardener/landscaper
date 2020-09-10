package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

func func_uniq(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("uniq takes exactly 1 argument")
	}

	list, ok := arguments[0].([]yaml.Node)
	if !ok {
		return info.Error("invalid type for function uniq")
	}

	var newList []yaml.Node

	for _, v := range list {
		found := false
		for _, n := range newList {
			r, _, _ := compareEquals(v.Value(), n.Value())
			if r {
				found = true
				break
			}
		}
		if !found {
			newList = append(newList, v)
		}
	}
	return newList, info, true
}
