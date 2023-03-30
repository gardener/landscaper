package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	RegisterFunction("features", func_features)
}

func func_features(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	switch len(arguments) {
	case 0:
		result := []yaml.Node{}
		for f := range binding.GetFeatures() {
			result = append(result, NewNode(f, binding))
		}
		return result, info, true
	case 1:
		name, ok := arguments[0].(string)
		if !ok {
			return info.Error("features: argument must be a string")
		}
		return binding.GetFeatures().Enabled(name), info, true
	default:
		return info.Error("features acctepts a maximum of one arguments")
	}
}
