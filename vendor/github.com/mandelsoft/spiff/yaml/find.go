package yaml

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/features"
)

var listIndex = regexp.MustCompile(`^\[(-?\d+)\]$`)

func Find(root Node, features features.FeatureFlags, path ...string) (Node, bool) {
	return FindR(false, root, features, path...)
}

func FindR(raw bool, root Node, features features.FeatureFlags, path ...string) (Node, bool) {
	here := root

	for _, step := range path {
		if here == nil {
			return nil, false
		}

		var found bool

		here, found = nextStep(raw, step, here, features)
		if !found {
			return nil, false
		}
	}

	return here, true
}

func FindString(root Node, features features.FeatureFlags, path ...string) (string, bool) {
	return FindStringR(false, root, features, path...)
}

func FindStringR(raw bool, root Node, features features.FeatureFlags, path ...string) (string, bool) {
	node, ok := FindR(raw, root, features, path...)
	if !ok {
		debug.Debug("%v not found", path)
		return "", false
	}

	val, ok := node.Value().(string)
	return val, ok
}

func FindInt(root Node, features features.FeatureFlags, path ...string) (int64, bool) {
	return FindIntR(false, root, features, path...)
}

func FindIntR(raw bool, root Node, features features.FeatureFlags, path ...string) (int64, bool) {
	node, ok := FindR(raw, root, features, path...)
	if !ok {
		return 0, false
	}

	val, ok := node.Value().(int64)
	return val, ok
}

func nextStep(raw bool, step string, here Node, features features.FeatureFlags) (Node, bool) {
	found := false

	switch v := here.Value().(type) {
	case map[string]Node:
		if !raw && !IsMapResolved(v, features) {
			return nil, false
		}
		here, found = v[step]
	case []Node:
		if !raw && !IsListResolved(v, features) {
			return nil, false
		}
		here, found = stepThroughList(raw, v, step, here.KeyName(), features)
	default:
	}

	return here, found
}

func stepThroughList(raw bool, here []Node, step string, key string, features features.FeatureFlags) (Node, bool) {
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

		name, ok := FindStringR(raw, sub, features, key)
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

func IsMapResolved(m map[string]Node, features features.FeatureFlags) bool {
	if features != nil && features.ControlEnabled() {
		for k := range m {
			if strings.HasPrefix(k, "<<") {
				return false
			}
		}
	}
	return m["<<"] == nil && m[MERGEKEY] == nil
}

func IsListResolved(l []Node, features features.FeatureFlags) bool {
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
