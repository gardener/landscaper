package dynaml

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

type UnresolvedNodes struct {
	Nodes []UnresolvedNode
}

type UnresolvedNode struct {
	yaml.Node

	Context []string
	Path    []string
}

func PrintableNodeValue(node yaml.Node) interface{} {
	nv := node.Value()
	switch nv.(type) {
	case map[string]yaml.Node:
		nv = "<map>"
	case []yaml.Node:
		nv = "<list>"
	}
	return nv
}

func (e UnresolvedNodes) Issue(msgfmt string, args ...interface{}) (result yaml.Issue, localError bool, failed bool) {
	format := ""
	result = yaml.NewIssue(msgfmt, args...)
	localError = false
	failed = false

	for _, node := range e.Nodes {
		issue := node.Issue()
		msg := issue.Issue
		if msg != "" {
			msg = tag(node) + msg
		}
		if node.HasError() {
			localError = true
		}
		if node.Failed() {
			failed = true
		}
		switch node.Value().(type) {
		case Expression:
			format = "(( %s ))\tin %s\t%s\t(%s)%s"
		default:
			format = "%s\tin %s\t%s\t(%s)%s"
		}
		nv := PrintableNodeValue(node)
		message := fmt.Sprintf(
			format,
			nv,
			node.SourceName(),
			strings.Join(node.Context, "."),
			strings.Join(node.Path, "."),
			msg,
		)
		issue.Issue = message
		result.Nested = append(result.Nested, issue)
	}
	return
}

func (e UnresolvedNodes) HasError() bool {
	for _, node := range e.Nodes {
		issue := node.Issue()
		msg := issue.Issue
		if msg != "" {
			return true
		}
	}
	return false
}

func (e UnresolvedNodes) Error() string {
	message := "unresolved nodes:"
	format := ""

	for _, node := range e.Nodes {
		issue := node.Issue()
		msg := issue.Issue
		if msg != "" {
			msg = "\t" + tag(node) + msg
		}
		nv := PrintableNodeValue(node)
		switch nv.(type) {
		case Expression:
			format = "%s\n\t(( %s ))\tin %s\t%s\t(%s)%s"
		default:
			format = "%s\n\t%s\tin %s\t%s\t(%s)%s"
		}
		val := strings.Replace(fmt.Sprintf("%s", nv), "\n", "\n\t", -1)
		message = fmt.Sprintf(
			format,
			message,
			val,
			node.SourceName(),
			strings.Join(node.Context, "."),
			strings.Join(node.Path, "."),
			msg,
		)
		message += nestedIssues(issue)
	}

	return message
}

func tag(node yaml.Node) string {
	tag := " "
	if !node.Failed() {
		tag = "@"
	} else {
		tag = "-"
	}
	if node.HasError() {
		tag = "*"
	}
	return tag
}

func nestedIssues(issue yaml.Issue) string {
	prefix := "\n"
	gap := ""
	if issue.Sequence {
		prefix = "\n\t\t... "
	} else {
		gap = "\t\t\t"
	}
	message := ""
	if issue.Nested != nil {
		for _, sub := range issue.Nested {
			message = message + prefix + sub.Issue
			message += nestedIssues(sub)
		}
	}
	return strings.Replace(message, "\n", "\n"+gap, -1)
}

func FindUnresolvedNodes(root yaml.Node, context ...string) (result []UnresolvedNode) {
	if root == nil {
		return result
	}

	var nodes []UnresolvedNode
	dummy := []string{"dummy"}
	found := false

	switch val := root.Value().(type) {
	case map[string]yaml.Node:
		for key, val := range val {
			nodes = append(
				nodes,
				FindUnresolvedNodes(val, addContext(context, key)...)...,
			)
		}

	case []yaml.Node:
		for i, val := range val {
			context := addContext(context, fmt.Sprintf("[%d]", i))

			nodes = append(
				nodes,
				FindUnresolvedNodes(val, context...)...,
			)
		}

	case Expression:
		nodes = append(nodes, UnresolvedNode{
			Node:    root,
			Context: context,
			Path:    effectivePath(root, context),
		})
		found = true

	case TemplateValue:
		//		context := addContext(context, fmt.Sprintf("&"))

		//		nodes = append(
		//			nodes,
		//			FindUnresolvedNodes(val.Orig, context...)...,
		//		)

	case string:
		if s := yaml.EmbeddedDynaml(root, false); s != nil {
			_, err := Parse(*s, dummy, dummy)
			if err != nil {
				nodes = append(nodes, UnresolvedNode{
					Node:    yaml.IssueNode(root, true, false, yaml.Issue{Issue: err.Error()}),
					Context: context,
					Path:    []string{},
				})
				found = true
			}
		}
	}

	if root.Failed() {
		if !found {
			nodes = append(nodes, UnresolvedNode{
				Node:    root,
				Context: context,
				Path:    effectivePath(root, context),
			})
		}
	}

	for _, n := range nodes {
		if n.GetAnnotation().HasError() {
			result = append(result, n)
		}
	}
	for _, n := range nodes {
		if !n.GetAnnotation().HasError() && !n.GetAnnotation().Failed() {
			result = append(result, n)
		}
	}
	for _, n := range nodes {
		if !n.GetAnnotation().HasError() && n.GetAnnotation().Failed() {
			result = append(result, n)
		}
	}
	return result
}

func ResetUnresolvedNodes(root yaml.Node) yaml.Node {
	if root == nil {
		return root
	}

	switch elem := root.Value().(type) {
	case map[string]yaml.Node:
		for key, val := range elem {
			elem[key] = ResetUnresolvedNodes(val)
		}

	case []yaml.Node:
		for i, val := range elem {
			elem[i] = ResetUnresolvedNodes(val)
		}

	case Expression:
		root = NewNode(fmt.Sprintf("(( %s ))", elem), nil)
	}

	return root
}

func effectivePath(node yaml.Node, context []string) []string {
	var path []string
	switch val := node.Value().(type) {
	case AutoExpr:
		path = val.Path
	case MergeExpr:
		path = val.Path
	default:
		orig := node.Issue().OrigPath
		if orig != nil {
			if !reflect.DeepEqual(context, orig) {
				if len(orig) > len(context) && reflect.DeepEqual(context, orig[:len(context)]) {
					path = append(append(orig[0:0:0], ".."), orig[len(context):]...)
				} else {
					path = orig
				}
			}
		}
	}
	return path
}

func addContext(context []string, step string) []string {
	dup := make([]string, len(context))
	copy(dup, context)
	return append(dup, step)
}

func IsExpression(val interface{}) bool {
	if val == nil {
		return false
	}
	_, ok := val.(Expression)
	return ok
}

func isLocallyResolved(node yaml.Node, binding Binding) bool {
	return isLocallyResolvedValue(node.Value(), binding)
}

func isLocallyResolvedValue(value interface{}, binding Binding) bool {
	switch v := value.(type) {
	case Expression:
		return false
	case map[string]yaml.Node:
		if !yaml.IsMapResolved(v, binding.GetFeatures()) {
			return false
		}
	case []yaml.Node:
		if !yaml.IsListResolved(v, binding.GetFeatures()) {
			return false
		}
	default:
	}

	return true
}

func IsResolvedNode(node yaml.Node, binding Binding) bool {
	if node == nil {
		return false
	}
	if node.Failed() || node.Undefined() {
		return false
	}
	return isResolvedValue(node.Value(), binding)
}

func isResolved(node yaml.Node, binding Binding) bool {
	return node == nil || isResolvedValue(node.Value(), binding)
}

func _isResolved(node yaml.Node, acceptFailed bool, binding Binding) bool {
	if node == nil || (acceptFailed && (node.Failed() || node.HasError())) {
		return true
	}
	return _isResolvedValue(node.Value(), acceptFailed, binding)
}

func isResolvedValue(val interface{}, binding Binding) bool {
	return _isResolvedValue(val, false, binding)
}

func _isResolvedValue(val interface{}, acceptFailed bool, binding Binding) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case Expression:
		return false
	case []yaml.Node:
		for _, n := range v {
			if !_isResolved(n, acceptFailed, binding) {
				return false
			}
		}
		return true
	case map[string]yaml.Node:
		if !yaml.IsMapResolved(v, binding.GetFeatures()) {
			return false
		}
		for _, n := range v {
			if !_isResolved(n, acceptFailed, binding) {
				return false
			}
		}
		return true

	case string:
		if yaml.EmbeddedDynaml(NewNode(val, nil), false) != nil {
			return false
		}
		return true
	default:
		return true
	}
}
