package dynaml

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/mandelsoft/spiff/yaml"
	"path/filepath"
)

func func_lookup(directory bool, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if !binding.GetState().FileAccessAllowed() {
		return info.DenyOSOperation("lookup")
	}

	paths := []string{}

	switch len(arguments) {
	case 0, 1:
		return info.Error("lookup_file requires at least two arguments")
	default:
		for index, arg := range arguments[1:] {
			switch v := arg.(type) {
			case []yaml.Node:
				for _, p := range v {
					if p.Value() == nil {
						continue
					}
					switch v := p.Value().(type) {
					case string:
						paths = append(paths, v)
					default:
						return info.Error("lookup_file: argument %d must be a list of strings", index)
					}
				}
			case string:
				paths = append(paths, v)
			default:
				return info.Error("lookup_file: argument %d must be a string or a list of strings", index)
			}
		}
	}

	name, ok := arguments[0].(string)
	if !ok {
		return info.Error("lookup_file: first argument must be a string")
	}

	if name == "" {
		return info.Error("lookup_file: first argument is empty string")
	}

	result := []yaml.Node{}
	if filepath.IsAbs(name) {
		if checkExistence(binding, name, directory) {
			result = append(result, NewNode(name, binding))
		}
		return result, info, true
	}

	for _, d := range paths {
		if d != "" {
			p := d + "/" + name
			if checkExistence(binding, p, directory) {
				result = append(result, NewNode(p, binding))
			}
		}
	}
	return result, info, true
}

func checkExistence(binding Binding, path string, directory bool) bool {
	if !binding.GetState().FileAccessAllowed() {
		return false
	}
	s, err := binding.GetState().FileSystem().Stat(path)
	if vfs.IsErrNotExist(err) || err != nil {
		return false
	}
	return s.IsDir() == directory
}
