package dynaml

import (
	"fmt"
	"sort"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type MappingExpr struct {
	A       Expression
	Lambda  Expression
	Context MappingContext
}

func (e MappingExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true
	inline := isInline(e.Lambda)
	debug.Debug("evaluate mapping\n")
	value, info, ok := ResolveExpressionOrPushEvaluation(&e.A, &resolved, nil, binding, true)
	if !ok {
		return nil, info, false
	}
	debug.Debug("MAP EXPR with resolver %+v\n", binding)
	lvalue, infoe, ok := ResolveExpressionOrPushEvaluation(&e.Lambda, &resolved, nil, binding, false)
	if !ok {
		return nil, info, false
	}

	if !resolved {
		return e, info.Join(infoe), ok
	}

	lambda, ok := lvalue.(LambdaValue)
	if !ok {
		return infoe.Error("mapping requires a lambda value")
	}

	debug.Debug("map: using lambda %+v\n", lambda)
	var result interface{}
	switch value.(type) {
	case []yaml.Node:
		if e.Context.Supports(value) {
			result, info, ok = mapList(inline, value.([]yaml.Node), lambda, binding, e.Context.CreateMappingAggregation())
		} else {
			return info.Error("list value not supported for %s mapping", e.Context.Keyword())
		}

	case map[string]yaml.Node:
		if e.Context.Supports(value) {
			result, info, ok = mapMap(inline, value.(map[string]yaml.Node), lambda, binding, e.Context.CreateMappingAggregation())
		} else {
			return info.Error("map value not supported for %s mapping", e.Context.Keyword())
		}

	default:
		return info.Error("map or list required for %s mapping", e.Context.Keyword())
	}
	if !ok {
		return nil, info, false
	}
	if result == nil {
		return e, info, true
	}
	debug.Debug("map: --> %+v\n", result)
	return result, info, true
}

func (e MappingExpr) String() string {
	lambda, ok := e.Lambda.(LambdaExpr)
	b := e.Context.Brackets()
	if ok {
		return fmt.Sprintf("%s%c%s%s%c", e.Context.Keyword(), b[0], e.A, fmt.Sprintf("%s", lambda)[len("lambda"):], b[1])
	} else {
		return fmt.Sprintf("%s%c%s|%s%c", e.Context.Keyword(), b[0], e.A, e.Lambda, b[1])
	}
}

///////////////////////////////////////////////////////////////////////////////
// handler context
///////////////////////////////////////////////////////////////////////////////

type MappingContext interface {
	CreateMappingAggregation() MappingAggregation
	Keyword() string
	Brackets() string
	Supports(source interface{}) bool
}

type MappingAggregation interface {
	Map(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo)
	Result() interface{}
}

type defaultContext struct {
	brackets string
	keyword  string
	list     bool
}

func (c *defaultContext) Keyword() string {
	return c.keyword
}
func (c *defaultContext) Brackets() string {
	return c.brackets
}
func (c *defaultContext) Supports(source interface{}) bool {
	switch source.(type) {
	case map[string]yaml.Node:
		return true
	case []yaml.Node:
		return c.list
	default:
		return false
	}
}

///////////////////////////////////////////////////////////////////////////////
//  map to list context
///////////////////////////////////////////////////////////////////////////////

type mapToListContext struct {
	defaultContext
}

var MapToListContext = &mapToListContext{defaultContext{brackets: "[]", keyword: "map", list: true}}

func (c *mapToListContext) CreateMappingAggregation() MappingAggregation {
	return &mapToList{[]yaml.Node{}}
}

type mapToList struct {
	result []yaml.Node
}

func (m *mapToList) Map(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) {
	if info.Undefined {
		return
	}
	if value == nil {
		return
	}
	m.result = append(m.result, NewNode(value, info))
}

func (m *mapToList) Result() interface{} {
	return m.result
}

///////////////////////////////////////////////////////////////////////////////
//  map to map context
///////////////////////////////////////////////////////////////////////////////

type mapToMapContext struct {
	defaultContext
}

var MapToMapContext = &mapToMapContext{defaultContext{brackets: "{}", keyword: "map", list: false}}

func (c *mapToMapContext) CreateMappingAggregation() MappingAggregation {
	return &mapToMap{map[string]yaml.Node{}}
}

type mapToMap struct {
	result map[string]yaml.Node
}

func (m *mapToMap) Map(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) {
	if info.Undefined {
		return
	}
	if value == nil {
		return
	}
	m.result[key.(string)] = NewNode(value, info)
}

func (m *mapToMap) Result() interface{} {
	return m.result
}

///////////////////////////////////////////////////////////////////////////////
// global handler functions
///////////////////////////////////////////////////////////////////////////////

func mapList(inline bool, source []yaml.Node, e LambdaValue, binding Binding, aggr MappingAggregation) (interface{}, EvaluationInfo, bool) {
	inp := make([]interface{}, len(e.lambda.Parameters))
	info := DefaultInfo()

	if len(e.lambda.Parameters) > 2 {
		info.Error("mapping expression takes a maximum of 2 arguments")
		return nil, info, false
	}
	for i, n := range source {
		debug.Debug("map:  mapping for %d: %+v\n", i, n)
		inp[0] = i
		inp[len(inp)-1] = n.Value()
		resolved, mapped, info, ok := e.Evaluate(inline, false, false, nil, inp, binding, false)
		if !ok {
			debug.Debug("map:  %d %+v: failed\n", i, n)
			return nil, info, false
		}
		if !resolved {
			return nil, info, ok
		}
		_, ok = mapped.(Expression)
		if ok {
			debug.Debug("map:  %d unresolved  -> KEEP\n", i)
			return nil, info, true
		}
		debug.Debug("map:  %d --> %+v\n", i, mapped)
		aggr.Map(i, mapped, n, info)
	}
	return aggr.Result(), info, true
}

func mapMap(inline bool, source map[string]yaml.Node, e LambdaValue, binding Binding, aggr MappingAggregation) (interface{}, EvaluationInfo, bool) {
	inp := make([]interface{}, len(e.lambda.Parameters))
	info := DefaultInfo()

	keys := getSortedKeys(source)
	for _, k := range keys {
		n := source[k]
		debug.Debug("map:  mapping for %s: %+v\n", k, n)
		inp[0] = k
		inp[len(inp)-1] = n.Value()
		resolved, mapped, info, ok := e.Evaluate(inline, false, false, nil, inp, binding, false)
		if !ok {
			debug.Debug("map:  %s %+v: failed\n", k, n)
			return nil, info, false
		}
		if !resolved {
			return nil, info, ok
		}
		_, ok = mapped.(Expression)
		if ok {
			debug.Debug("map:  %s unresolved  -> KEEP\n", k)
			return nil, info, true
		}
		debug.Debug("map:  %s --> %+v\n", k, mapped)
		aggr.Map(k, mapped, n, info)
	}
	return aggr.Result(), info, true
}

func getSortedKeys(unsortedMap map[string]yaml.Node) []string {
	keys := make([]string, len(unsortedMap))
	i := 0
	for k, _ := range unsortedMap {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}
