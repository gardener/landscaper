package dynaml

import (
	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

func func_list_to_map(listexpr Expression, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var list []yaml.Node

	_, info, _ := listexpr.Evaluate(binding, false)

	key := info.KeyName
	if key == "" {
		key = "name"
	}

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("list_to_map takes 1 or 2 arguments")
	}

	switch v := arguments[0].(type) {
	case []yaml.Node:
		list = v
	default:
		return info.Error("list_to_map requires a list as first argument")
	}

	if len(arguments) == 2 {
		switch v := arguments[1].(type) {
		case string:
			key = v
		default:
			return info.Error("second argument of list_to_map must be a key name")
		}
	}

	debug.Debug("list to map with key field '%s'", key)

	result, err := listToMap(list, key)
	if result == nil {
		return info.Error(err, key)
	}
	return result, info, true
}

func listToMap(list []yaml.Node, keyName string) (map[string]yaml.Node, string) {
	toMap := make(map[string]yaml.Node)

	for _, val := range list {
		asMap, ok := val.Value().(map[string]yaml.Node)
		if !ok {
			return nil, "list entries must by maps"
		}
		keyValue, ok := asMap[keyName]
		if !ok {
			return nil, "key field '%s' not found"
		}
		key, ok := keyValue.Value().(string)
		if !ok {
			return nil, "key field '%s' contains no string value"
		}
		newMap := make(map[string]yaml.Node)
		for key, val := range asMap {
			if key != keyName {
				newMap[key] = val
			}
		}

		toMap[key] = yaml.SubstituteNode(newMap, val)
	}

	return toMap, ""
}
