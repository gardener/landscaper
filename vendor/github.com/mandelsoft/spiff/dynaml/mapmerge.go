package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/yaml"
)

func func_merge(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 1 {
		return info.Error("at least one argument required for merge function")
	}

	args := []yaml.Node{}

	if len(arguments) == 1 {
		l, ok := arguments[0].([]yaml.Node)
		if ok {
			for i, e := range l {
				m, err := getMap(i, e.Value())
				if err != nil {
					return info.Error("merge: entry of list argument: %s", err)
				}
				args = append(args, yaml.NewNode(m, "dynaml"))
			}
			if len(args) == 0 {
				return info.Error("merge: no map found for merge")
			}
		}
	}
	if len(args) == 0 {
		for i, arg := range arguments {
			m, err := getMap(i, arg)
			if err != nil {
				return info.Error("merge: argument %s", err)
			}
			args = append(args, yaml.NewNode(m, "dynaml"))
		}
	}
	result, err := binding.Cascade(binding, args[0], false, args[1:]...)
	if err != nil {
		info.SetError("merging failed: %s", err)
		return nil, info, false
	}

	return result.Value(), info, true
}

func getMap(n int, arg interface{}) (map[string]yaml.Node, error) {
	temp, ok := arg.(TemplateValue)
	if ok {
		arg, ok := node_copy(temp.Prepared).Value().(map[string]yaml.Node)
		if !ok {
			return nil, fmt.Errorf("%d: template is not a map template", n+1)
		}
		return arg, nil
	}
	m, ok := arg.(map[string]yaml.Node)
	if ok {
		return m, nil
	}
	return nil, fmt.Errorf("%d: no map or map template, but %s", n+1, ExpressionType(arg))
}
