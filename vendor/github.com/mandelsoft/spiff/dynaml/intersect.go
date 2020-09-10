package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	RegisterFunction("intersect", func_intersect)
}

func func_intersect(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var result []yaml.Node
	info := DefaultInfo()

	for i, l := range arguments {
		switch alist := l.(type) {
		case []yaml.Node:
			if result == nil {
				result = alist
			} else {
				newList := []yaml.Node{}
				for _, e := range result {
					found := false
					for _, n := range alist {
						r, _, _ := compareEquals(e.Value(), n.Value())
						if r {
							found = true
							break
						}
					}
					if found {
						newList = append(newList, e)
					}
				}
				result = newList
			}
		case nil:
		default:
			return info.Error("intersect: argument %d: type '%s'(%s) cannot be intersected", i, ExpressionType(l))
		}
	}

	return result, info, true
}
