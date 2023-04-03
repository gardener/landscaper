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
	if !e.Context.Supports(value) {
		return info.Error("%s%s does not support %s values", e.Context.Keyword(), e.Context.Brackets(), ExpressionType(value))
	}
	result, info, ok = e.Context.CreateMappingAggregation(value).DoMapping(inline, value, lambda, binding)

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
	CreateMappingAggregation(source interface{}) MappingAggregation
	Keyword() string
	Brackets() string
	Supports(source interface{}) bool
}

type MappingAggregation interface {
	DoMapping(inline bool, value interface{}, e LambdaValue, binding Binding) (interface{}, EvaluationInfo, bool)
	Add(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) error
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

func (c *mapToListContext) CreateMappingAggregation(source interface{}) MappingAggregation {
	return &mapToList{MapperForSource(source), []yaml.Node{}}
}

type mapToList struct {
	mapper Mapper
	result []yaml.Node
}

func (m *mapToList) DoMapping(inline bool, value interface{}, e LambdaValue, binding Binding) (interface{}, EvaluationInfo, bool) {
	return m.mapper(inline, value, e, binding, m)
}

func (m *mapToList) Add(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) error {
	if info.Undefined {
		return nil
	}
	if value == nil {
		return nil
	}
	m.result = append(m.result, NewNode(value, info))
	return nil
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

func (c *mapToMapContext) CreateMappingAggregation(source interface{}) MappingAggregation {
	switch source.(type) {
	case map[string]yaml.Node:
		return &mapMapToMap{map[string]yaml.Node{}}
	case []yaml.Node:
		return &mapListToMap{map[string]yaml.Node{}}
	default:
		return nil
	}
}

func (c *mapToMapContext) Supports(source interface{}) bool {
	switch source.(type) {
	case map[string]yaml.Node:
		return true
	case []yaml.Node:
		return true
	default:
		return false
	}
}

///////////////////////////////////////////////////////////////////////////////
//  map map to map

type mapMapToMap struct {
	result map[string]yaml.Node
}

func (m *mapMapToMap) DoMapping(inline bool, value interface{}, e LambdaValue, binding Binding) (interface{}, EvaluationInfo, bool) {
	return mapMap(inline, value, e, binding, m)
}

func (m *mapMapToMap) Add(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) error {
	if info.Undefined {
		return nil
	}
	if value == nil {
		return nil
	}
	m.result[key.(string)] = NewNode(value, info)
	return nil
}

func (m *mapMapToMap) Result() interface{} {
	return m.result
}

///////////////////////////////////////////////////////////////////////////////
//  map list to map

type mapListToMap struct {
	result map[string]yaml.Node
}

func (m *mapListToMap) DoMapping(inline bool, value interface{}, e LambdaValue, binding Binding) (interface{}, EvaluationInfo, bool) {
	return mapList(inline, value, e, binding, m)
}

func (m *mapListToMap) Add(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) error {
	if info.Undefined {
		return nil
	}
	if value == nil {
		return nil
	}
	if s, ok := n.Value().(string); ok {
		m.result[s] = NewNode(value, info)
		return nil
	}
	return fmt.Errorf("list element must be string, but found %s", ExpressionType(n.Value()))
}

func (m *mapListToMap) Result() interface{} {
	return m.result
}

///////////////////////////////////////////////////////////////////////////////
// global handler functions
///////////////////////////////////////////////////////////////////////////////

type Mapper func(inline bool, value interface{}, e LambdaValue, binding Binding, aggr MappingAggregation) (interface{}, EvaluationInfo, bool)

func MapperForSource(value interface{}) Mapper {
	switch value.(type) {
	case []yaml.Node:
		return mapList
	case map[string]yaml.Node:
		return mapMap
	}
	return nil
}

func mapList(inline bool, value interface{}, e LambdaValue, binding Binding, aggr MappingAggregation) (interface{}, EvaluationInfo, bool) {
	source := value.([]yaml.Node)
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
		err := aggr.Add(i, mapped, n, info)
		if err != nil {
			return info.Error("%s", err)
		}
	}
	return aggr.Result(), info, true
}

func mapMap(inline bool, value interface{}, e LambdaValue, binding Binding, aggr MappingAggregation) (interface{}, EvaluationInfo, bool) {
	source := value.(map[string]yaml.Node)
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
		err := aggr.Add(k, mapped, n, info)
		if err != nil {
			return info.Error("%s", err)
		}
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
