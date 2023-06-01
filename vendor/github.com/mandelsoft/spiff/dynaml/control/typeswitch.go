package control

import (
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	dynaml.RegisterControl("type", flowType, "default")
}

func flowType(ctx *dynaml.ControlContext) (yaml.Node, bool) {
	if node, ok := dynaml.ControlReady(ctx, true); !ok {
		return node, false
	}
	return selected(ctx, dynaml.ExpressionType(ctx.Value))
}
