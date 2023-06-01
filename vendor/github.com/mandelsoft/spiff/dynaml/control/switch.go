package control

import (
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/yaml"
)

func init() {
	dynaml.RegisterControl("switch", flowSwitch, "default", "cases")
}

func flowSwitch(ctx *dynaml.ControlContext) (yaml.Node, bool) {
	if ctx.Value.Undefined() {
		return yaml.UndefinedNode(dynaml.NewNode(nil, ctx)), true
	}
	cases := ctx.Option("cases")
	if node, ok := dynaml.ControlReady(ctx, cases == nil); !ok {
		return node, false
	}
	if cases != nil {
		switch list := cases.Value().(type) {
		case []yaml.Node:
			for i, c := range list {
				if m, ok := c.Value().(map[string]yaml.Node); ok {
					var value yaml.Node
					reject := false
					found := false
					for k, v := range m {
						switch k {
						case "case":
							found = true
							reject = reject || !v.EquivalentToNode(ctx.Value)
						case "match":
							found = true
							if f, ok := v.Value().(dynaml.LambdaValue); ok {
								resolved, r, _, ok := f.Evaluate(false, false, false, nil, []interface{}{ctx.Value.Value()}, ctx, false)
								if !ok {
									reject = true
								} else {
									if !resolved {
										return ctx.Node, false
									}
									if b, ok := r.(bool); !ok {
										return dynaml.ControlIssue(ctx, "case %d boolean match result required, but got %s", i, dynaml.ExpressionType(r))
									} else {
										reject = reject || !b
									}
								}
							}
						case "value":
							value = v
						default:
							return dynaml.ControlIssue(ctx, "case %d invalid field %q", i, k)
						}
					}
					if !found {
						return dynaml.ControlIssue(ctx, "case %d requires 'case' or `'match' field", i)
					}
					if !reject {
						if value != nil {
							return dynaml.ControlValue(ctx, value)
						} else {
							return yaml.UndefinedNode(dynaml.NewNode(nil, ctx)), true
						}
					}
				} else {
					return dynaml.ControlIssue(ctx, "case %d requires field 'case'", i)
				}
			}
			result := ctx.Option("default")
			if result != nil {
				return dynaml.ControlValue(ctx, result)
			}
			return dynaml.ControlIssue(ctx, "invalid switch value: %s", dynaml.Shorten(dynaml.Short(ctx.Value.Value(), false)))
		default:
			return dynaml.ControlIssue(ctx, "cases must be a list")
		}
	}
	return selected(ctx, ctx.Value.Value())
}

func selected(ctx *dynaml.ControlContext, key interface{}) (yaml.Node, bool) {
	var result yaml.Node
	if key != nil {
		switch v := key.(type) {
		case string:
			result = ctx.Field(v)
		default:
			return dynaml.ControlIssue(ctx, "invalid switch value type: %s", dynaml.ExpressionType(v))
		}
	}
	if result == nil {
		result = ctx.Option("default")
	}
	if result != nil {
		return dynaml.ControlValue(ctx, result)
	}
	return dynaml.ControlIssue(ctx, "invalid switch value: %q", key)
}
