// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"sync"

	errors "github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	KIND_OPTIONTYPE = "option type"
	KIND_OPTION     = "option"
)

type OptionTypeCreator func(name string, description string) OptionType

type ValueTypeInfo struct {
	OptionTypeCreator
	Description string
}

func (i ValueTypeInfo) GetDescription() string {
	return i.Description
}

type Registry = *registry

var DefaultRegistry = New()

type registry struct {
	lock        sync.RWMutex
	valueTypes  map[string]ValueTypeInfo
	optionTypes map[string]OptionType
}

func New() Registry {
	return &registry{
		valueTypes:  map[string]ValueTypeInfo{},
		optionTypes: map[string]OptionType{},
	}
}

func (r *registry) RegisterOptionType(t OptionType) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.optionTypes[t.GetName()] = t
}

func (r *registry) RegisterValueType(name string, c OptionTypeCreator, desc string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.valueTypes[name] = ValueTypeInfo{OptionTypeCreator: c, Description: desc}
}

func (r *registry) GetValueType(name string) *ValueTypeInfo {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if t, ok := r.valueTypes[name]; ok {
		return &t
	}
	return nil
}

func (r *registry) GetOptionType(name string) OptionType {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.optionTypes[name]
}

func (r *registry) CreateOptionType(typ, name, desc string) (OptionType, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	t, ok := r.valueTypes[typ]
	if !ok {
		return nil, errors.ErrUnknown(KIND_OPTIONTYPE, typ)
	}

	n := t.OptionTypeCreator(name, desc)
	o := r.optionTypes[name]
	if o != nil {
		if o.ValueType() != n.ValueType() {
			return nil, errors.ErrAlreadyExists(KIND_OPTION, name)
		}
		return o, nil
	}
	return n, nil
}

func (r *registry) Usage() string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	tinfo := utils.FormatMap("", r.valueTypes)
	oinfo := utils.FormatMap("", r.optionTypes)

	return `
The following predefined option types can be used:

` + oinfo + `

The following predefined value types are supported:

` + tinfo
}
