package dynaml

import (
	"fmt"
	"log"

	"github.com/mandelsoft/spiff/yaml"

	"github.com/mandelsoft/spiff/legacy/candiedyaml"
)

func func_format(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return format("format", arguments, binding)
}

func format(name string, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 1 {
		return info.Error("alt least one argument required for '%s'", name)
	}

	args := make([]interface{}, len(arguments))
	for i, arg := range arguments {
		switch v := arg.(type) {
		case []yaml.Node:
			yaml, err := candiedyaml.Marshal(NewNode(v, nil))
			if err != nil {
				log.Fatalln("error marshalling yaml fragment:", err)
			}
			args[i] = string(yaml)
		case map[string]yaml.Node:
			yaml, err := candiedyaml.Marshal(NewNode(v, nil))
			if err != nil {
				log.Fatalln("error marshalling yaml fragment:", err)
			}
			args[i] = string(yaml)
		case TemplateValue:
			yaml, err := candiedyaml.Marshal(v.Orig)
			if err != nil {
				log.Fatalln("error marshalling template:", err)
			}
			args[i] = string(yaml)
		case LambdaValue:
			args[i] = v.String()
		default:
			args[i] = arg
		}
	}

	f, ok := args[0].(string)
	if !ok {
		return info.Error("%s: format must be string", format)
	}
	return fmt.Sprintf(f, args[1:]...), info, true
}
