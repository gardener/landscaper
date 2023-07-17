package semver

import (
	"sort"

	"github.com/Masterminds/semver/v3"

	. "github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/yaml"
)

const F_Sort = "semversort"

func init() {
	RegisterFunction(F_Sort, func_sort)
}

func func_sort(args []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(args) == 1 {
		if a, ok := args[0].([]yaml.Node); ok {
			args = make([]interface{}, len(a))
			for i, v := range a {
				args[i] = v.Value()
			}
		}
	}
	versions := make([]*semver.Version, len(args))

	for i, a := range args {
		v, info := parse(F_Compare, a)
		if v == nil {
			return nil, info, false
		}
		versions[i] = v
	}
	sort.Sort(semver.Collection(versions))

	val := make([]yaml.Node, len(args))
	for i, v := range versions {
		val[i] = yaml.NewNode(v.Original(), binding.SourceName())
	}
	return val, info, true
}
