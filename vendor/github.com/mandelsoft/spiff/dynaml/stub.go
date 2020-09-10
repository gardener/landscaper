package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
	"strings"
)

func (e CallExpr) stub(binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	var arg []string

	switch len(e.Arguments) {
	case 0:
		arg = binding.Path()
	case 1:
		pushed := e.Arguments[0]
		ref, ok := pushed.(ReferenceExpr)
		if !ok {
			resolved := true
			val, info, ok := ResolveExpressionOrPushEvaluation(&pushed, &resolved, nil, binding, true)
			if !resolved {
				return e, info, true
			}

			if !ok {
				return val, info, ok
			} else {
				switch v := val.(type) {
				case string:
					arg = PathComponents(v, true)
				case []yaml.Node:
					for _, n := range v {
						str, ok := n.Value().(string)
						if !ok {
							return info.Error("stub() requires a string entries in list")
						}
						arg = append(arg, str)
					}
				default:
					return info.Error("stub() requires a string or reference argument")
				}
			}
		} else {
			arg = ref.Path
		}

	default:
		return info.Error("a maximum of one argument expected for 'stub'")
	}

	stub, ok := binding.FindInStubs(arg)
	if !ok {
		return info.Error("'%s' not found in any stub", strings.Join(arg, "."))
	}
	return stub.Value(), info, ok
}
