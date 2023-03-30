package flow

import (
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/yaml"

	_ "github.com/mandelsoft/spiff/dynaml/control"
)

func flowControl(node yaml.Node, undef map[string]yaml.Node, env dynaml.Binding) (yaml.Node, bool, bool) {
	flags := node.GetAnnotation().Flags()
	resolved := false
	is := false
	ctx, err := dynaml.GetControl(node, undef, env)
	if err != nil {
		node, resolved = dynaml.ControlIssue(ctx, err.Error())
	} else if ctx != nil {
		if err == nil {
			is = true
			node, resolved = ctx.Function()
		}
	}
	if resolved {
		if flags != 0 {
			node = yaml.AddFlags(node, flags)
		}
	}
	return node, is, resolved
}
