package dynaml

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

type ControlContext struct {
	Binding
	*Control
	Value   yaml.Node
	Node    yaml.Node
	Fields  map[string]yaml.Node
	Options map[string]yaml.Node
}

func (c *ControlContext) Option(name string) yaml.Node {
	return c.Options[name]
}
func (c *ControlContext) Field(name string) yaml.Node {
	return c.Fields[name]
}
func (c *ControlContext) HasFields() bool {
	return len(c.Fields) != 0
}
func (c *ControlContext) SortedFields() []string {
	return yaml.GetSortedKeys(c.Fields)
}
func (c *ControlContext) DefinedFields() map[string]yaml.Node {
	result := map[string]yaml.Node{}
	for k, v := range c.Fields {
		if !v.Undefined() {
			result[k] = v
		}
	}
	return result
}
func (c *ControlContext) Function() (yaml.Node, bool) {
	return c.Control.Function(c)
}

type ControlFunction func(*ControlContext) (yaml.Node, bool)

type Control struct {
	name     string
	options  map[string]bool
	function ControlFunction
}

func (c *Control) Name() string {
	return c.name
}

func (c *Control) Options() []string {
	l := []string{}
	for n := range c.options {
		l = append(l, n)
	}
	return l
}

func (c *Control) HasOption(name string) bool {
	_, ok := c.options[name]
	return ok
}

func (c *Control) IsTemplateOption(name string) bool {
	return c.options[name]
}

func (c *Control) Function(env *ControlContext) (yaml.Node, bool) {
	env.Control = c
	return c.function(env)
}

type Controls interface {
	RegisterControl(name string, f ControlFunction, opts ...string) error
	LookupControl(name string) (*Control, bool)
	IsTemplateControlOption(name string) bool
}

type controlRegistry struct {
	controls        map[string]*Control
	templateoptions map[string]struct{}
}

func newControls() *controlRegistry {
	return &controlRegistry{map[string]*Control{}, map[string]struct{}{}}
}

func NewControls() Controls {
	r := newControls()

	for n, c := range control_registry.controls {
		r.controls[n] = c
	}
	for n := range control_registry.templateoptions {
		r.templateoptions[n] = struct{}{}
	}
	return r
}

func (r *controlRegistry) RegisterControl(name string, f ControlFunction, opts ...string) error {
	m := map[string]bool{}
	for _, o := range opts {
		t := false
		if strings.HasPrefix(o, "*") {
			t = true
			o = o[1:]
		}
		m[o] = t

		if _, ok := r.templateoptions[o]; ok && !t {
			return fmt.Errorf("ambigious control option template setting for %q", o)
		}
		if t {
			r.templateoptions[o] = struct{}{}
		}
	}
	c := &Control{
		name:     name,
		options:  m,
		function: f,
	}
	if _, ok := r.controls[c.name]; ok {
		return fmt.Errorf("control or option %q already defined", c.name)
	}
	r.controls[c.name] = c
	for _, o := range opts {
		if strings.HasPrefix(o, "*") {
			o = o[1:]
		}
		if old, ok := r.controls[o]; ok {
			if old != nil {
				return fmt.Errorf("option %q for control %q already defined as control", o, c.name)
			}
		}
		r.controls[o] = nil
	}
	return nil
}

func (r *controlRegistry) LookupControl(name string) (*Control, bool) {
	c, ok := r.controls[name]
	return c, ok
}

func (r *controlRegistry) IsTemplateControlOption(name string) bool {
	_, ok := r.templateoptions[name]
	return ok
}

var control_registry = newControls()

func RegisterControl(name string, f ControlFunction, opts ...string) {
	err := control_registry.RegisterControl(name, f, opts...)
	if err != nil {
		panic(err.Error())
	}
}

func ControlIssue(ctx *ControlContext, msg string, args ...interface{}) (yaml.Node, bool) {
	var issue yaml.Issue
	if len(args) == 0 {
		issue = yaml.NewIssue("%s", msg)
	} else {
		issue = yaml.NewIssue(msg, args...)
	}
	return ControlIssueByIssue(ctx, issue, true)
}

func ControlIssueByIssue(ctx *ControlContext, issue yaml.Issue, final bool) (yaml.Node, bool) {
	control := "<control>"
	if ctx.Control != nil {
		control = fmt.Sprintf("<%s control>", ctx.name)
	}
	if !final {
		return yaml.IssueNode(ctx.Node, true, true, issue), false
	}
	return yaml.IssueNode(yaml.NewNode(control, ctx.Node.SourceName()), true, true, issue), false
}

func IsControl(node yaml.Node, env Binding) (bool, error) {
	c, err := GetControl(node, nil, env)
	return c != nil, err
}

func RequireTemplate(opt string, env Binding) bool {
	registry := env.GetState().GetRegistry()
	if strings.HasPrefix(opt, "<<") {
		return registry.IsTemplateControlOption(opt[2:])
	}
	return false
}

func GetControl(node yaml.Node, undef map[string]yaml.Node, env Binding) (*ControlContext, error) {
	if env.GetFeatures().ControlEnabled() {
		if m, ok := node.Value().(map[string]yaml.Node); ok {
			registry := env.GetState().GetRegistry()
			var name string
			var val yaml.Node
			var control *Control
			opts := map[string]yaml.Node{}
			fields := map[string]yaml.Node{}

			f := func(k string, v yaml.Node) error {
				if strings.HasPrefix(k, "<<") {
					n := k[2:]
					if n != "" && n != "<" && n[0] != '!' {
						c, ok := registry.LookupControl(n)
						if !ok {
							return fmt.Errorf("unknown control or control option %q", k)
						}
						if c != nil {
							if control != nil {
								return fmt.Errorf("multiple controls %q and %q", name, k)
							}
							name = k
							control = c
							val = v
						} else {
							opts[n] = v
						}
					}
					return nil
				}
				fields[k] = v
				return nil
			}

			for k, v := range m {
				if err := f(k, v); err != nil {
					return nil, err
				}
			}
			for k, v := range undef {
				if err := f(k, v); err != nil {
					return nil, err
				}
			}

			if control != nil {
				for k, v := range undef {
					m[k] = v
				}
				if err := control.CheckOpts(opts); err != nil {
					return nil, err
				}
				return &ControlContext{
					Binding: env,
					Control: control,
					Node:    node,
					Value:   val,
					Fields:  fields,
					Options: opts,
				}, nil
			} else {
				if len(opts) > 0 {
					return nil, fmt.Errorf("control options %v without control", yaml.GetSortedKeys(opts))
				}
			}
		}
	}
	return nil, nil
}

func (c *Control) CheckOpts(opts map[string]yaml.Node) error {
	for o := range opts {
		if _, ok := c.options[o]; !ok {
			return fmt.Errorf("invalid option %q for control %q", o, c.name)
		}
	}
	return nil
}

func ControlValue(ctx *ControlContext, val yaml.Node) (yaml.Node, bool) {
	if val.Undefined() || IsResolvedNode(val, ctx) {
		return val, true
	}
	return val, false
}

func ControlReady(ctx *ControlContext, acceptFields bool) (yaml.Node, bool) {
	if !acceptFields && ctx.HasFields() {
		return ControlIssue(ctx, "no regular fields %v allowed", ctx.SortedFields())
	}
	return ctx.Node, (ctx.Value.Undefined() || IsResolvedNode(ctx.Value, ctx)) && _isResolvedValue(ctx.Options, true, ctx) && _isResolvedValue(ctx.Fields, true, ctx)
}
