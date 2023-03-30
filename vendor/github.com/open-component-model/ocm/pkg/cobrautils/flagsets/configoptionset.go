// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsets

import (
	"fmt"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type ConfigOptionType interface {
	GetName() string
	GetDescription() string

	Create() Option

	Equal(optionType ConfigOptionType) bool
}

type ConfigOptionTypeSet interface {
	AddGroups(groups ...string)

	GetName() string

	Size() int
	OptionTypes() []ConfigOptionType
	OptionTypeNames() []string
	SharedOptionTypes() []ConfigOptionType

	HasOptionType(name string) bool
	HasSharedOptionType(name string) bool

	GetSharedOptionType(name string) ConfigOptionType
	GetOptionType(name string) ConfigOptionType
	GetTypeSet(name string) ConfigOptionTypeSet
	OptionTypeSets() []ConfigOptionTypeSet

	AddOptionType(ConfigOptionType) error
	AddTypeSet(ConfigOptionTypeSet) error
	AddAll(o ConfigOptionTypeSet) (duplicated ConfigOptionTypeSet, err error)

	Close(funcs ...func([]ConfigOptionType) error) error

	CreateOptions() ConfigOptions
	AddGroupsToOption(o Option)
}

type configOptionTypeSet struct {
	lock    sync.RWMutex
	name    string
	options map[string]ConfigOptionType
	sets    map[string]ConfigOptionTypeSet
	shared  map[string][]ConfigOptionTypeSet
	groups  []string

	closed bool
}

func NewConfigOptionSet(name string, types ...ConfigOptionType) ConfigOptionTypeSet {
	set := &configOptionTypeSet{
		name:    name,
		options: map[string]ConfigOptionType{},
		sets:    map[string]ConfigOptionTypeSet{},
		shared:  map[string][]ConfigOptionTypeSet{},
	}
	for _, t := range types {
		set.AddOptionType(t)
	}
	return set
}

func (s *configOptionTypeSet) AddGroups(groups ...string) {
	s.groups = AddGroups(s.groups, groups...)
}

func (s *configOptionTypeSet) Close(funcs ...func([]ConfigOptionType) error) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if len(funcs) > 0 {
		list := s.optionTypes()
		for _, f := range funcs {
			if f != nil {
				err := f(list)
				if err != nil {
					return err
				}
			}
		}
	}
	s.closed = true
	return nil
}

func (s *configOptionTypeSet) GetName() string {
	return s.name
}

func (s *configOptionTypeSet) AddOptionType(optionType ConfigOptionType) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return errors.ErrClosed("config option set")
	}
	name := optionType.GetName()
	s.options[name] = optionType
	return nil
}

func (s *configOptionTypeSet) Size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.options)
}

func (s *configOptionTypeSet) OptionTypes() []ConfigOptionType {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.optionTypes()
}

func (s *configOptionTypeSet) optionTypes() []ConfigOptionType {
	var list []ConfigOptionType
	for _, o := range s.options {
		list = append(list, o)
	}
	return list
}

func (s *configOptionTypeSet) OptionTypeNames() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return utils.StringMapKeys(s.options)
}

func (s *configOptionTypeSet) SharedOptionTypes() []ConfigOptionType {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var list []ConfigOptionType
	for n, o := range s.options {
		if _, ok := s.shared[n]; ok {
			list = append(list, o)
		}
	}
	return list
}

func (s *configOptionTypeSet) HasOptionType(name string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	_, ok := s.options[name]
	return ok
}

func (s *configOptionTypeSet) HasSharedOptionType(name string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	_, ok := s.shared[name]
	return ok
}

func (s *configOptionTypeSet) GetOptionType(name string) ConfigOptionType {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.options[name]
}

func (s *configOptionTypeSet) GetSharedOptionType(name string) ConfigOptionType {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.shared[name]; ok {
		return s.options[name]
	}
	return nil
}

func (s *configOptionTypeSet) AddTypeSet(set ConfigOptionTypeSet) error {
	if set == nil {
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return errors.ErrClosed("config option set")
	}

	name := set.GetName()
	if nested, ok := s.sets[name]; ok {
		if nested == set {
			return nil
		}
		return fmt.Errorf("%s: config type set with name %q already added", s.GetName(), name)
	}

	return set.Close(func(list []ConfigOptionType) error {
		// check for problem first
		err := s.check(list)
		if err != nil {
			return err
		}
		// now align data structure
		for _, o := range list {
			if _, ok := s.options[o.GetName()]; ok {
				s.shared[o.GetName()] = append(s.shared[o.GetName()], set)
			} else {
				s.options[o.GetName()] = o
			}
		}
		s.sets[name] = set
		return nil
	})
}

func (s *configOptionTypeSet) GetTypeSet(name string) ConfigOptionTypeSet {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.sets[name]
}

func (s *configOptionTypeSet) OptionTypeSets() []ConfigOptionTypeSet {
	s.lock.RLock()
	defer s.lock.RUnlock()

	result := make([]ConfigOptionTypeSet, 0, len(s.sets))
	for _, t := range s.sets {
		result = append(result, t)
	}
	return result
}

func (s *configOptionTypeSet) AddGroupsToOption(o Option) {
	if !s.HasOptionType(o.GetName()) {
		return
	}
	if len(s.groups) > 0 {
		o.AddGroups(s.groups...)
	}
	for _, set := range s.shared[o.GetName()] {
		set.AddGroupsToOption(o)
	}
}

func (s *configOptionTypeSet) CreateOptions() ConfigOptions {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var opts []Option

	for n := range s.options {
		opt := s.options[n].Create()
		s.AddGroupsToOption(opt)
		opts = append(opts, opt)
	}
	return NewOptions(opts)
}

func (s *configOptionTypeSet) AddAll(o ConfigOptionTypeSet) (duplicates ConfigOptionTypeSet, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return nil, errors.ErrClosed("config option set")
	}

	list := o.OptionTypes()
	if err := s.check(list); err != nil {
		return nil, err
	}
	duplicates = NewConfigOptionSet("duplicates")
	for _, t := range list {
		_, ok := s.options[t.GetName()]
		if !ok {
			s.options[t.GetName()] = t
		} else {
			duplicates.AddOptionType(t)
		}
	}
	return duplicates, nil
}

func (s *configOptionTypeSet) check(list []ConfigOptionType) error {
	for _, o := range list {
		old := s.options[o.GetName()]
		if old != nil && !old.Equal(o) {
			return fmt.Errorf("option type %s doesn't match (%T<->%T)", o.GetName(), o, old)
		}
	}
	return nil
}
