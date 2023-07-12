package dynaml

import (
	"path"
	"regexp"
	"strings"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

var templ_pattern = regexp.MustCompile(".*\\s+&template(\\(?|\\s+).*")

func func_read(cached bool, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) > 2 {
		return info.Error("read takes a maximum of two arguments")
	}
	if !binding.GetState().FileAccessAllowed() {
		return info.DenyOSOperation("read")
	}

	file, ok := arguments[0].(string)
	if !ok {
		return info.Error("string value required for file path")
	}

	t := "text"
	if strings.HasSuffix(file, ".yml") || strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".json") {
		t = "yaml"
	}
	if len(arguments) > 1 {
		t, ok = arguments[1].(string)
		if !ok {
			return info.Error("string value required for type")
		}

	}

	data, err := binding.GetFileContent(file, cached)
	if err != nil {
		return info.Error("read: %s", err)
	}
	return ParseData(file, data, t, binding)
}

func ParseData(file string, data []byte, mode string, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	info.Source = file

	rerooted := binding
	if strings.HasPrefix(mode, ".") {
		rerooted = binding.WithNewRoot()
		mode = mode[1:]
	}
	switch mode {
	case "template":
		n, err := yaml.Parse(file, data)
		if err != nil {
			return info.Error("error parsing file [%s]: %s", path.Clean(file), err)
		}
		return asTemplate(n, binding), info, true

	case "templates":
		nodes, err := yaml.ParseMulti(file, data)
		if err != nil {
			return info.Error("error parsing file [%s]: %s", path.Clean(file), err)
		}
		result := []yaml.Node{}
		for _, n := range nodes {
			result = append(result, NewNode(asTemplate(n, binding), n))
		}
		return result, info, true
	case "yaml":
		node, err := yaml.Parse(file, data)
		if err != nil {
			return info.Error("error parsing file [%s]: %s", path.Clean(file), err)
		}
		debug.Debug("resolving yaml file\n")
		result, state := rerooted.Flow(node, false)
		if state != nil {
			debug.Debug("resolving yaml file failed: %s", state.Error())
			return info.PropagateError(nil, state, "resolution of yaml file '%s' failed", file)
		}
		debug.Debug("resolving yaml file succeeded")
		return result.Value(), info, true
	case "multiyaml":
		nodes, err := yaml.ParseMulti(file, data)
		if err != nil {
			return info.Error("error parsing file [%s]: %s", path.Clean(file), err)
		}
		for len(nodes) > 1 && nodes[len(nodes)-1].Value() == nil {
			nodes = nodes[:len(nodes)-1]
		}
		debug.Debug("resolving yaml list from file\n")
		result, state := rerooted.Flow(NewNode(nodes, info), false)
		if state != nil {
			debug.Debug("resolving yaml file failed: " + state.Error())
			return info.PropagateError(nil, state, "resolution of yaml file '%s' failed", file)
		}
		debug.Debug("resolving yaml file succeeded")
		return result.Value(), info, true
	case "import":
		node, err := yaml.Parse(file, data)
		if err != nil {
			return info.Error("error parsing file [%s]: %s", path.Clean(file), err)
		}
		info.Raw = true
		debug.Debug("import yaml file succeeded")
		return node.Value(), info, true
	case "importmulti":
		nodes, err := yaml.ParseMulti(file, data)
		if err != nil {
			return info.Error("error parsing file [%s]: %s", path.Clean(file), err)
		}
		info.Raw = true
		for len(nodes) > 1 && nodes[len(nodes)-1].Value() == nil {
			nodes = nodes[:len(nodes)-1]
		}
		return nodes, info, true

	case "text":
		return string(data), info, true

	case "binary":
		return Base64Encode(data, 60), info, true

	default:
		return info.Error("invalid file type [%s] %s", path.Clean(file), mode)
	}
}

func asTemplate(n yaml.Node, binding Binding) TemplateValue {
	orig := node_copy(n)

	switch v := orig.Value().(type) {
	case map[string]yaml.Node:
		if _, ok := v[yaml.MERGEKEY]; !ok {
			v[yaml.MERGEKEY] = NewNode("(( &template ))", n)
		} else {
			if _, ok := v["<<"]; !ok {
				v["<<"] = NewNode("(( &template ))", n)
			}
		}
	case []yaml.Node:
		found := false
		for _, e := range v {
			if m, ok := e.Value().(map[string]yaml.Node); ok {
				e, ok := m[yaml.MERGEKEY]
				if !ok {
					e, ok = m["<<"]
				}
				if ok {
					s := yaml.EmbeddedDynaml(e, binding.GetState().InterpolationEnabled())
					if s != nil && templ_pattern.MatchString(*s) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			new := []yaml.Node{NewNode(map[string]yaml.Node{yaml.MERGEKEY: NewNode("(( &template ))", n)}, n)}
			new = append(new, v...)
			orig = NewNode(new, n)
		}
	}
	return NewTemplateValue(binding.Path(), n, orig, binding)
}
