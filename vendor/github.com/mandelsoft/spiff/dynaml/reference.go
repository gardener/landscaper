package dynaml

import (
	"strings"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type ReferenceExpr struct {
	Path []string
}

func (e ReferenceExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	fromRoot := e.Path[0] == ""

	debug.Debug("reference: %v\n", e.Path)
	return e.find(func(end int, path []string) (yaml.Node, bool) {
		if fromRoot {
			return binding.FindFromRoot(path[1 : end+1])
		} else {
			return binding.FindReference(path[:end+1])
		}
	}, binding, locally)
}

func (e ReferenceExpr) String() string {
	return strings.Join(e.Path, ".")
}

func (e ReferenceExpr) find(f func(int, []string) (node yaml.Node, x bool), binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	var step yaml.Node
	var ok bool

	info := DefaultInfo()
	debug.Debug("resolving ref [%v]", e.Path)
	for i := 0; i < len(e.Path); i++ {
		step, ok = f(i, e.Path)

		debug.Debug("  %d: %v %+v\n", i, ok, step)
		if !ok {
			return info.Error("'%s' not found", strings.Join(e.Path[0:i+1], "."))
		}

		if !isLocallyResolved(step) {
			debug.Debug("  locally unresolved %T\n", step.Value())
			if _, ok := step.Value().(Expression); ok {
				info.Issue = yaml.NewIssue("'%s' unresolved", strings.Join(e.Path[0:i+1], "."))
			} else {
				info.Issue = yaml.NewIssue("'%s' not complete", strings.Join(e.Path[0:i+1], "."))
			}
			info.Failed = step.Failed() || step.HasError()
			return e, info, true
		}
	}

	if !locally && !isResolvedValue(step.Value()) {
		debug.Debug("  unresolved\n")
		info.Issue = yaml.NewIssue("'%s' unresolved", strings.Join(e.Path, "."))
		info.Failed = step.Failed() || step.HasError()
		return e, info, true
	}

	debug.Debug("reference %v -> %+v\n", e.Path, step)
	info.KeyName = step.KeyName()
	return value(yaml.ReferencedNode(step)), info, true
}
