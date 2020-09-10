package yaml

import (
	"fmt"
	"reflect"
)

func Equals(a, b Node, path []string) (bool, string) {

	if reflect.TypeOf(a.Value()) != reflect.TypeOf(b.Value()) {
		return false, fmt.Sprintf("non matching type %T!=%T at %+v", a.Value(), b.Value(), path)

	}
	if a.Value() == nil && b.Value() == nil {
		return true, ""
	}
	if a.Value() == nil || b.Value() == nil {
		return false, fmt.Sprintf("found non-matching nil at %+v", path)
	}

	if !reflect.DeepEqual(a.GetAnnotation(), b.GetAnnotation()) {
		return false, fmt.Sprintf("annotation diff at %+v: %v", path, a.GetAnnotation())
	}
	switch va := a.Value().(type) {
	case []Node:
		vb := b.Value().([]Node)
		if len(va) != len(vb) {
			return false, fmt.Sprintf("list length mismatch %d!=%d at %+v", len(va), len(vb), path)
		}
		for i, v := range va {
			if b, r := Equals(v, vb[i], append(path, fmt.Sprintf("[%d]", i))); !b {
				return b, r
			}
		}
	case map[string]Node:
		vb := b.Value().(map[string]Node)

		for k, v := range va {
			_, ok := vb[k]
			if !ok {
				return false, fmt.Sprintf("key %q missing in b at %+v", k, path)
			}
			if b, r := Equals(v, vb[k], append(path, k)); !b {
				return b, r
			}
		}
		for k := range vb {
			_, ok := va[k]
			if !ok {
				return false, fmt.Sprintf("additional key %q in b at %+v", k, path)
			}
		}
	default:
		e := reflect.DeepEqual(a.Value(), b.Value())
		if !e {
			return false, fmt.Sprintf("diff at %+v: %v!=%v", path, a.Value(), b.Value())
		}

	}
	return true, ""
}
