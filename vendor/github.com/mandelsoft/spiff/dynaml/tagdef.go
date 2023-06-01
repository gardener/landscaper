package dynaml

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

const F_TagDef = "tagdef"

func init() {
	RegisterFunction(F_TagDef, func_tagdef)
}

func func_tagdef(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	ttype := "local"

	if len(arguments) < 2 || len(arguments) > 3 {
		return info.Error("two or three arguments expected for %s", F_TagDef)
	}
	name, ok := arguments[0].(string)
	if !ok {
		return info.Error("name argument for %s must be a string", F_TagDef)
	}
	if strings.HasPrefix(name, "*") && len(arguments) == 2 {
		name = name[1:]
		ttype = "global"
	}
	if err := CheckTagName(name); err != nil {
		return info.Error("invalid tag name %q for %s: %s", name, F_TagDef, err)
	}
	value := yaml.NewNode(arguments[1], fmt.Sprintf("tagdef(%s)", binding.Path()))
	if len(arguments) == 3 {
		str, ok := arguments[2].(string)
		if !ok {
			return info.Error("type argument for %s must be a string (local or global)", F_TagDef)
		}
		ttype = str
	}
	var scope TagScope
	switch ttype {
	case "local":
		scope = TAG_SCOPE_STREAM
	case "global":
		scope = TAG_SCOPE_GLOBAL
	default:
		return info.Error("invalid scope argument %q for %s (must be local or global)", ttype, F_TagDef)
	}
	scope |= TAG_LOCAL
	err := binding.GetState().SetTag(name, value, binding.Path(), scope)
	if err != nil {
		return info.Error("%s", err)
	}
	return arguments[1], info, true
}
