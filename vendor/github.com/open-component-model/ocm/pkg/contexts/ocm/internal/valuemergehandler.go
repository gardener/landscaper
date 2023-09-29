// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sync"

	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type ValueMergeHandlerSpecification = metav1.MergeAlgorithmSpecification

type ValueMergeHandlerConfig interface {
	Complete(ctx Context) error
}

type ValueMergeHandler interface {
	Algorithm() string
	Description() string
	DecodeConfig(data []byte) (ValueMergeHandlerConfig, error)

	Merge(ctx Context, src runtime.RawValue, tgt *runtime.RawValue, cfg ValueMergeHandlerConfig) (bool, error)
}

type ValueMergeHandlerRegistry interface {
	RegisterHandler(h ValueMergeHandler)
	AssignHandler(hint string, spec *ValueMergeHandlerSpecification)

	GetHandler(name string) ValueMergeHandler
	GetAssignment(typ string) *ValueMergeHandlerSpecification

	GetHandlers() MergeHandlers
	GetAssignments() MergeHandlerAssignments

	Copy() ValueMergeHandlerRegistry
}

type mergeHandlerRegistry struct {
	lock sync.Mutex
	base ValueMergeHandlerRegistry

	handlerTypes MergeHandlers
	assignments  MergeHandlerAssignments
}

var _ ValueMergeHandlerRegistry = (*mergeHandlerRegistry)(nil)

func NewValueMergeHandlerRegistry(base ...ValueMergeHandlerRegistry) ValueMergeHandlerRegistry {
	return &mergeHandlerRegistry{
		base:         utils.Optional(base...),
		handlerTypes: MergeHandlers{},
		assignments:  MergeHandlerAssignments{},
	}
}

func (m *mergeHandlerRegistry) RegisterHandler(h ValueMergeHandler) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.handlerTypes[h.Algorithm()] = h
}

func (m *mergeHandlerRegistry) AssignHandler(hint string, spec *ValueMergeHandlerSpecification) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.assignments[hint] = spec
}

func (m *mergeHandlerRegistry) GetHandler(algo string) ValueMergeHandler {
	m.lock.Lock()
	defer m.lock.Unlock()
	h := m.handlerTypes[algo]
	if h == nil && m.base != nil {
		return m.base.GetHandler(algo)
	}
	return h
}

func (m *mergeHandlerRegistry) GetAssignment(hint string) *ValueMergeHandlerSpecification {
	m.lock.Lock()
	defer m.lock.Unlock()
	h := m.assignments[hint]
	if h == nil && m.base != nil {
		return m.base.GetAssignment(hint)
	}
	return h
}

func (m *mergeHandlerRegistry) Copy() ValueMergeHandlerRegistry {
	m.lock.Lock()
	defer m.lock.Unlock()
	c := &mergeHandlerRegistry{
		base:         m.base,
		handlerTypes: map[string]ValueMergeHandler{},
		assignments:  map[string]*ValueMergeHandlerSpecification{},
	}
	for k, v := range m.handlerTypes {
		c.handlerTypes[k] = v
	}
	for k, v := range m.assignments {
		c.assignments[k] = v
	}
	return c
}

type MergeHandlers = map[string]ValueMergeHandler

func (m *mergeHandlerRegistry) GetHandlers() MergeHandlers {
	m.lock.Lock()
	defer m.lock.Unlock()

	r := MergeHandlers{}
	if m.base != nil {
		r = m.base.GetHandlers()
	}
	for k, v := range m.handlerTypes {
		r[k] = v
	}
	return r
}

type MergeHandlerAssignments = map[string]*ValueMergeHandlerSpecification

func (m *mergeHandlerRegistry) GetAssignments() MergeHandlerAssignments {
	m.lock.Lock()
	defer m.lock.Unlock()

	r := MergeHandlerAssignments{}
	if m.base != nil {
		r = m.base.GetAssignments()
	}
	for k, v := range m.assignments {
		r[k] = v
	}
	return r
}

////////////////////////////////////////////////////////////////////////////////

var DefaultValueMergeHandlerRegistry = NewValueMergeHandlerRegistry()
