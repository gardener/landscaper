package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

///////////////////////////////////////////////////////////////////////////////
// select to list context
///////////////////////////////////////////////////////////////////////////////

type selectToListContext struct {
	defaultContext
}

var SelectToListContext = &selectToListContext{defaultContext{brackets: "[]", keyword: "select", list: true}}

func (c *selectToListContext) CreateMappingAggregation(source interface{}) MappingAggregation {
	return &selectToList{MapperForSource(source), []yaml.Node{}}
}

type selectToList struct {
	mapper Mapper
	result []yaml.Node
}

func (m *selectToList) DoMapping(inline bool, value interface{}, e LambdaValue, binding Binding) (interface{}, EvaluationInfo, bool) {
	return m.mapper(inline, value, e, binding, m)
}

func (m *selectToList) Add(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) error {
	if info.Undefined || value == nil || !toBool(value) {
		return nil
	}
	m.result = append(m.result, n)
	return nil
}

func (m *selectToList) Result() interface{} {
	return m.result
}

///////////////////////////////////////////////////////////////////////////////
// select to map context
///////////////////////////////////////////////////////////////////////////////

type selectToMapContext struct {
	defaultContext
}

var SelectToMapContext = &selectToMapContext{defaultContext{brackets: "{}", keyword: "select", list: false}}

func (c *selectToMapContext) CreateMappingAggregation(source interface{}) MappingAggregation {
	return &selectToMap{MapperForSource(source), map[string]yaml.Node{}}
}

type selectToMap struct {
	mapper Mapper
	result map[string]yaml.Node
}

func (m *selectToMap) DoMapping(inline bool, value interface{}, e LambdaValue, binding Binding) (interface{}, EvaluationInfo, bool) {
	return m.mapper(inline, value, e, binding, m)
}

func (m *selectToMap) Add(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) error {
	if info.Undefined || value == nil || !toBool(value) {
		return nil
	}
	m.result[key.(string)] = n
	return nil
}

func (m *selectToMap) Result() interface{} {
	return m.result
}
