package control

import (
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	dynaml.RegisterControl("if", flowIf, "then", "else")
}

func flowIf(ctx *dynaml.ControlContext) (yaml.Node, bool) {
	if node, ok := dynaml.ControlReady(ctx, false); !ok {
		return node, false
	}
	if ctx.Value.Value() == nil {
		if e := ctx.Option("else"); e != nil {
			return dynaml.ControlValue(ctx, e)
		}
		return yaml.UndefinedNode(dynaml.NewNode(nil, ctx)), true
	}
	switch v := ctx.Value.Value().(type) {
	case bool:
		if v {
			if e := ctx.Option("then"); e != nil {
				return dynaml.ControlValue(ctx, e)
			}
			return yaml.UndefinedNode(dynaml.NewNode(nil, ctx)), true
		} else {
			if e := ctx.Option("else"); e != nil {
				return dynaml.ControlValue(ctx, e)
			}
			return yaml.UndefinedNode(dynaml.NewNode(nil, ctx)), true
		}
	default:
		return dynaml.ControlIssue(ctx, "invalid condition value type: %s", dynaml.ExpressionType(v))
	}
}
