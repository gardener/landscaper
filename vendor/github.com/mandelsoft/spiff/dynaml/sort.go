package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
	"sort"
)

func func_sort(arguments []interface{}, binding Binding) (result interface{}, info EvaluationInfo, ok bool) {
	info = DefaultInfo()

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("sort takes one or two arguments")
	}

	list, ok := arguments[0].([]yaml.Node)
	if !ok {
		return info.Error("argument for sort must be a list")
	}

	var less Less

	if len(arguments) == 2 {
		lambda, ok := arguments[1].(LambdaValue)
		if !ok {
			return info.Error("second argument for sort must be a lambda function")
		}
		less = LambdaLess(lambda, list, binding)
	} else {
		less = ValueLess(list)
	}

	defer CatchEvaluationError(&result, &info, &ok, "sort failed")

	sort.SliceStable(list, less)
	return list, info, true
}

type Less func(i, j int) bool

func ValueLess(list []yaml.Node) Less {
	return func(i, j int) bool {
		switch a := list[i].Value().(type) {
		case string:
			b, ok := list[j].Value().(string)
			if ok {
				return a < b
			}
		case int64:
			b, ok := list[j].Value().(int64)
			if ok {
				return a < b
			}
		}
		RaiseEvaluationErrorf("list elements must either be strings or integers")
		return false
	}
}

func LambdaLess(lambda LambdaValue, list []yaml.Node, binding Binding) Less {
	return func(i, j int) bool {
		inp := []interface{}{list[i].Value(), list[j].Value()}
		resolved, v, info, ok := lambda.Evaluate(false, false, false, nil, inp, binding, false)
		if !ok || !resolved {
			RaiseEvaluationError(resolved, info, ok)
		}
		b, ok := v.(bool)
		if !ok {
			i, ok := v.(int64)
			if !ok {
				RaiseEvaluationErrorf("lambda must return a bool or integer")
			}
			b = i < 0
		}
		return b
	}
}
