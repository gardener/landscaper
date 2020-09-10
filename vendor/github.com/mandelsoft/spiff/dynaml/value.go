package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/yaml"
	"strconv"
)

type ValueExpr struct {
	Value interface{}
}

func (e ValueExpr) String() string {
	return ValueAsString(e.Value, true)
}

func (e ValueExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	return e.Value, DefaultInfo(), true
}

func ValueAsString(val interface{}, all bool) string {
	switch v := val.(type) {
	case []yaml.Node:
		s := "["
		sep := ""
		for _, e := range v {
			s = fmt.Sprintf("%s%s%s", s, sep, ValueAsString(e.Value(), all))
			sep = ", "
		}
		return s + "]"
	case map[string]yaml.Node:
		s := "{"
		sep := ""
		for _, k := range getSortedKeys(v) {
			if all || k != "_" {
				s = fmt.Sprintf("%s%s\"%s\"=%s", s, sep, k, ValueAsString(v[k].Value(), all))
				sep = ", "
			}
		}
		return s + "}"
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%s", v)
	}
}
