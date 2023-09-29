// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsets

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/errors"
)

func GetField(config Config, names ...string) (interface{}, error) {
	var cur interface{} = config

	for i, n := range names {
		if cur == nil {
			return nil, nil
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%s is no map", strings.Join(names[:i], "."))
		}
		cur = m[n]
	}
	return cur, nil
}

func SetField(config Config, value interface{}, names ...string) error {
	var last Config
	var cur interface{} = config

	if config == nil {
		return fmt.Errorf("no map given")
	}
	for i, n := range names {
		if cur == nil {
			cur = Config{}
			last[names[i-1]] = cur
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s is no map", strings.Join(names[:i], "."))
		}
		if i == len(names)-1 {
			m[n] = value
			return nil
		}
		last = m
		cur = m[n]
	}
	return fmt.Errorf("no field path given")
}

type NameProvider interface {
	GetName() string
}

type OptionName string

func (n OptionName) GetName() string {
	return string(n)
}

// OptionValueProvider provides values for named options.
type OptionValueProvider interface {
	GetValue(name string) (interface{}, bool)
}

// AddFieldByOption sets the specified target field with the option value, if given.
// If no target field is specified the name of the option is used.
func AddFieldByOption(opts OptionValueProvider, oname string, config Config, names ...string) error {
	return AddFieldByMappedOption(opts, oname, config, nil, names...)
}

func AddFieldByMappedOption(opts OptionValueProvider, oname string, config Config, mapper func(interface{}) (interface{}, error), names ...string) error {
	var err error

	if v, ok := opts.GetValue(oname); ok {
		if len(names) == 0 {
			names = []string{oname}
		}
		if mapper != nil {
			v, err = mapper(v)
			if err != nil {
				return errors.Wrapf(err, "option %q", oname)
			}
		}
		return SetField(config, v, names...)
	}

	return nil
}

// AddFieldByOptionP sets the specified target field with the option value, if given.
// The option is specified by a name provider instead of its name.
// If no target field is specified the name of the option is used.
func AddFieldByOptionP(opts OptionValueProvider, p NameProvider, config Config, names ...string) error {
	return AddFieldByMappedOption(opts, p.GetName(), config, nil, names...)
}

func AddFieldByMappedOptionP(opts OptionValueProvider, p NameProvider, config Config, mapper func(interface{}) (interface{}, error), names ...string) error {
	return AddFieldByMappedOption(opts, p.GetName(), config, mapper, names...)
}

func ComposedAdder(adders ...ConfigAdder) ConfigAdder {
	switch len(adders) {
	case 0:
		return nil
	case 1:
		return adders[0]
	default:
		return func(opts ConfigOptions, config Config) error {
			for _, a := range adders {
				if a == nil {
					continue
				}
				if err := a(opts, config); err != nil {
					return err
				}
			}
			return nil
		}
	}
}

func AddGroups(list []string, groups ...string) []string {
outer:
	for _, g := range groups {
		for _, f := range list {
			if g == f {
				continue outer
			}
		}
		list = append(list, g)
	}
	return list
}
