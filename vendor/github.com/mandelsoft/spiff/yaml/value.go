package yaml

import (
	"sort"
)

type ComparableValue interface {
	EquivalentTo(interface{}) bool
}

func GetSortedKeys(unsortedMap map[string]Node) []string {
	keys := make([]string, len(unsortedMap))
	i := 0
	for k, _ := range unsortedMap {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}
