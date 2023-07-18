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

func (c *selectToListContext) CreateMappingAggregation() MappingAggregation {
	return &selectToList{[]yaml.Node{}}
}

type selectToList struct {
	result []yaml.Node
}

func newSelectToList() MappingAggregation {
	return &mapToList{[]yaml.Node{}}
}

func (m *selectToList) Map(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) {
	if info.Undefined || value == nil || !toBool(value) {
		return
	}
	m.result = append(m.result, n)
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

func (c *selectToMapContext) CreateMappingAggregation() MappingAggregation {
	return &selectToMap{map[string]yaml.Node{}}
}

type selectToMap struct {
	result map[string]yaml.Node
}

func (m *selectToMap) Map(key interface{}, value interface{}, n yaml.Node, info EvaluationInfo) {
	if info.Undefined || value == nil || !toBool(value) {
		return
	}
	m.result[key.(string)] = n
}

func (m *selectToMap) Result() interface{} {
	return m.result
}
