package control

import (
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	dynaml.RegisterControl("merge", flowMerge)
}

func flowMerge(ctx *dynaml.ControlContext) (yaml.Node, bool) {
	if node, ok := dynaml.ControlReady(ctx, true); !ok {
		return node, false
	}
	fields := ctx.DefinedFields()
	if ctx.Value.Value() != nil {
		switch v := ctx.Value.Value().(type) {
		case map[string]yaml.Node:
			for k, e := range v {
				fields[k] = e
			}
		case []yaml.Node:
			for i, l := range v {
				if l.Value() != nil {
					if m, ok := l.Value().(map[string]yaml.Node); ok {
						for k, e := range m {
							fields[k] = e
						}
					} else {
						return dynaml.ControlIssue(ctx, "entry %d: invalid entry type: %s", i, dynaml.ExpressionType(v))
					}
				}
			}

		default:
			if v != nil {
				return dynaml.ControlIssue(ctx, "invalid value type: %s", dynaml.ExpressionType(v))
			}
		}
	}
	return dynaml.NewNode(fields, ctx), true
}
