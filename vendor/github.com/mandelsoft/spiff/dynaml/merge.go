package dynaml

import (
	"github.com/mandelsoft/spiff/debug"
	"strings"
)

type MergeExpr struct {
	Path     []string
	Redirect bool
	Replace  bool
	Required bool
	KeyName  string
}

func (e MergeExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	var info EvaluationInfo
	info.KeyName = e.KeyName
	debug.Debug("/// lookup %v\n", e.Path)
	if e.Redirect {
		info.RedirectPath = e.Path
	}
	if len(e.Path) == 0 {
		info.Merged = true
		return nil, info, true
	}
	node, ok := binding.FindInStubs(e.Path)
	if ok {
		info.Replace = e.Replace
		info.Merged = true
		info.Source = node.SourceName()
		info.NodeFlags = node.Flags()
		return node.Value(), info, ok
	} else {
		return info.Error("'%s' not found in any stub", strings.Join(e.Path, "."))
	}
}

func (e MergeExpr) String() string {
	rep := ""
	if e.Replace {
		rep = " replace"
	}

	if e.KeyName != "" {
		rep += " on " + e.KeyName
	}
	if e.Required && !e.Redirect && rep != "" {
		rep = " required"
	}
	if e.Redirect {
		return "merge" + rep + " " + strings.Join(e.Path, ".")
	}
	return "merge" + rep
}
