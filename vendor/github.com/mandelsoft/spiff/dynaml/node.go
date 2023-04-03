package dynaml

import (
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

func value(node yaml.Node) interface{} {
	if node == nil {
		return nil
	}
	return node.Value()
}

func NewNode(val interface{}, src SourceProvider) yaml.Node {
	source := ""

	if src == nil || len(src.SourceName()) == 0 {
		source = "dynaml"
	} else {
		if !strings.HasPrefix(src.SourceName(), "dynaml@") {
			source = "dynaml@" + src.SourceName()
		} else {
			source = src.SourceName()
		}
	}

	return yaml.NewNode(val, source)
}

func IssueNode(env Binding, preservePath bool, node yaml.Node, error bool, failed bool, issue yaml.Issue) yaml.Node {
	if node.Issue().OrigPath != nil {
		issue.OrigPath = node.Issue().OrigPath
	} else {
		if env != nil && preservePath {
			issue.OrigPath = env.Path()
		}
	}
	return yaml.IssueNode(node, error, failed, issue)
}
