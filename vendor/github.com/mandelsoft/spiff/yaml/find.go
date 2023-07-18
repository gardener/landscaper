package yaml

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mandelsoft/spiff/debug"
)

var listIndex = regexp.MustCompile(`^\[(-?\d+)\]$`)

func Find(root Node, path ...string) (Node, bool) {
	return FindR(false, root, path...)
}

func FindR(raw bool, root Node, path ...string) (Node, bool) {
	here := root

	for _, step := range path {
		if here == nil {
			return nil, false
		}

		var found bool

		here, found = nextStep(raw, step, here)
		if !found {
			return nil, false
		}
	}

	return here, true
}

func FindString(root Node, path ...string) (string, bool) {
	return FindStringR(false, root, path...)
}

func FindStringR(raw bool, root Node, path ...string) (string, bool) {
	node, ok := FindR(raw, root, path...)
	if !ok {
		debug.Debug("%v not found", path)
		return "", false
	}

	val, ok := node.Value().(string)
	return val, ok
}

func FindInt(root Node, path ...string) (int64, bool) {
	return FindIntR(false, root, path...)
}

func FindIntR(raw bool, root Node, path ...string) (int64, bool) {
	node, ok := FindR(raw, root, path...)
	if !ok {
		return 0, false
	}

	val, ok := node.Value().(int64)
	return val, ok
}

func nextStep(raw bool, step string, here Node) (Node, bool) {
	found := false

	switch v := here.Value().(type) {
	case map[string]Node:
		if !raw && !IsMapResolved(v) {
			return nil, false
		}
		here, found = v[step]
	case []Node:
		if !raw && !IsListResolved(v) {
			return nil, false
		}
		here, found = stepThroughList(raw, v, step, here.KeyName())
	default:
	}

	return here, found
}

func stepThroughList(raw bool, here []Node, step string, key string) (Node, bool) {
	match := listIndex.FindStringSubmatch(step)
	if match != nil {
		index, err := strconv.Atoi(match[1])
		if err != nil {
			panic(err)
		}

		if index < 0 {
			index = len(here) + index
		}
		if len(here) <= index {
			return nil, false
		}

		return here[index], true
	}

	if key == "" {
		key = "name"
	}
	split := strings.Index(step, ":")
	if split > 0 {
		key = step[:split]
		step = step[split+1:]
	}

	for _, sub := range here {
		_, ok := sub.Value().(map[string]Node)
		if !ok {
			continue
		}

		name, ok := FindStringR(raw, sub, key)
		if !ok {
			continue
		}

		if name == step {
			return sub, true
		}
	}

	return nil, false
}

func PathComponent(step string) string {
	split := strings.Index(step, ":")
	if split > 0 {
		return step[split+1:]
	}
	return step
}

func UnresolvedListEntryMerge(node Node) (Node, string, bool) {
	subMap, ok := node.Value().(map[string]Node)
	if ok {
		if len(subMap) == 1 {
			inlineNode, ok := subMap["<<"]
			if ok {
				return inlineNode, "<<", true
			}
			inlineNode, ok = subMap[MERGEKEY]
			if ok {
				return inlineNode, MERGEKEY, true
			}
		}
	}
	return nil, "", false
}

func IsMapResolved(m map[string]Node) bool {
	return m["<<"] == nil && m[MERGEKEY] == nil
}

func IsListResolved(l []Node) bool {
	for _, val := range l {
		if val != nil {
			_, _, ok := UnresolvedListEntryMerge(val)
			if ok {
				return false
			}
		}
	}
	return true
}
