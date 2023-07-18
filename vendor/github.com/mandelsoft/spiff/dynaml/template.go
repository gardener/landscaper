package dynaml

import (
	"fmt"
	"path"
	"strings"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type SubstitutionExpr struct {
	Template Expression
}

func (e SubstitutionExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	debug.Debug("evaluating expression to determine template: %s\n", binding)
	n, info, ok := e.Template.Evaluate(binding, false)
	if !ok || isExpression(n) {
		return e, info, ok
	}
	inp := map[string]yaml.Node{}
	template, ok := n.(TemplateValue)
	if !ok {
		return info.Error("template value required")
	}
	prepared := node_copy(template.Prepared)
	inp[yaml.SELF] = yaml.ResolverNode(NewNode(n, binding), template.resolver)

	debug.Debug("resolving template '%s' %s\n", strings.Join(template.Path, "."), binding)
	result, state := binding.WithLocalScope(inp).Flow(prepared, false)
	info = DefaultInfo()
	if state != nil {
		if state.HasError() {
			debug.Debug("resolving template failed: " + state.Error())
			return info.PropagateError(e, state, "resolution of template '%s' failed", strings.Join(template.Path, "."))
		} else {
			debug.Debug("resolving template delayed: " + state.Error())
			return e, info, true
		}
	}
	debug.Debug("resolving template succeeded")
	info.Source = result.SourceName()
	return result.Value(), info, true
}

func (e SubstitutionExpr) String() string {
	return fmt.Sprintf("*(%s)", e.Template)
}

type TemplateValue struct {
	Path     []string
	Prepared yaml.Node
	Orig     yaml.Node
	resolver Binding
}

var _ StaticallyScopedValue = TemplateValue{}
var _ yaml.ComparableValue = TemplateValue{}

func NewTemplateValue(path []string, prepared yaml.Node, orig yaml.Node, binding Binding) TemplateValue {
	return TemplateValue{path, prepared, orig, binding}
}

func (e TemplateValue) String() string {
	return fmt.Sprintf("<template %s: %s>", path.Join(e.Path...), shorten(short(e.Prepared, false)))
}

func (e TemplateValue) MarshalYAML() (tag string, value interface{}, err error) {
	return e.Orig.MarshalYAML()
}

func (e TemplateValue) StaticResolver() Binding {
	return e.resolver
}

func (e TemplateValue) SetStaticResolver(binding Binding) StaticallyScopedValue {
	e.resolver = binding
	return e
}

func (e TemplateValue) EquivalentTo(val interface{}) bool {
	o, ok := val.(TemplateValue)
	return ok && e.Orig.EquivalentToNode(o.Orig)
}

func node_copy(node yaml.Node) yaml.Node {
	if node == nil {
		return nil
	}
	switch val := node.Value().(type) {
	case []yaml.Node:
		list := make([]yaml.Node, len(val))
		for i, v := range val {
			list[i] = node_copy(v)
		}
		return yaml.NewNode(list, node.SourceName())
	case map[string]yaml.Node:
		m := make(map[string]yaml.Node)
		for k, v := range val {
			m[k] = node_copy(v)
		}
		return yaml.NewNode(m, node.SourceName())
	}
	return yaml.NewNode(node.Value(), node.SourceName())
}
