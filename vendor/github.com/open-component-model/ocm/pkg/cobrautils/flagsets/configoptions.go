// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsets

import (
	"fmt"

	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"
)

type Option interface {
	GetName() string
	AddFlags(fs *pflag.FlagSet)
	Value() interface{}

	Changed() bool
	AddGroups(groups ...string)
}

type Filter func(name string) bool

type ConfigOptions interface {
	AddTypeSetGroupsToOptions(set ConfigOptionTypeSet)

	AddFlags(fs *pflag.FlagSet)

	Options() []Option

	Check(set ConfigOptionTypeSet, desc string) error
	GetValue(name string) (interface{}, bool)
	Changed(names ...string) bool

	FilterBy(Filter) ConfigOptions
}

func Not(f Filter) Filter {
	return func(name string) bool {
		return !f(name)
	}
}

type configOptions struct {
	options []Option
	flags   *pflag.FlagSet
}

func NewOptions(opts []Option) ConfigOptions {
	return &configOptions{options: opts}
}

func (o *configOptions) AddTypeSetGroupsToOptions(set ConfigOptionTypeSet) {
	for _, opt := range o.options {
		set.AddGroupsToOption(opt)
	}
}

func (o *configOptions) Options() []Option {
	return slices.Clone(o.options)
}

func (o *configOptions) GetValue(name string) (interface{}, bool) {
	for _, opt := range o.options {
		if opt.GetName() == name {
			return opt.Value(), o.flags.Changed(name)
		}
	}
	return nil, false
}

func (o *configOptions) AddFlags(fs *pflag.FlagSet) {
	for _, opt := range o.options {
		opt.AddFlags(fs)
	}
	o.flags = fs
}

func (o *configOptions) Changed(names ...string) bool {
	if len(names) == 0 {
		for _, opt := range o.options {
			if o.flags.Changed(opt.GetName()) {
				return true
			}
		}
		return false
	}

	set := map[string]struct{}{}
	for _, n := range names {
		set[n] = struct{}{}
	}
	for _, opt := range o.options {
		if _, ok := set[opt.GetName()]; ok {
			if o.flags.Changed(opt.GetName()) {
				return true
			}
		}
	}
	return false
}

func (o *configOptions) FilterBy(filter Filter) ConfigOptions {
	if filter == nil {
		return o
	}
	var options []Option

	for _, opt := range o.options {
		if filter(opt.GetName()) {
			options = append(options, opt)
		}
	}
	return &configOptions{
		options: options,
		flags:   o.flags,
	}
}

func (o *configOptions) Check(set ConfigOptionTypeSet, desc string) error {
	if desc != "" {
		desc = " for " + desc
	}

	if set == nil {
		for _, opt := range o.options {
			if o.flags.Changed(opt.GetName()) {
				return fmt.Errorf("option %q given, but not possible%s", opt.GetName(), desc)
			}
		}
	} else {
		for _, opt := range o.options {
			if o.flags.Changed(opt.GetName()) && set.GetOptionType(opt.GetName()) == nil {
				if desc == "" {
					return fmt.Errorf("option %q given, but not valid for option set %q", opt.GetName(), set.GetName())
				}
				return fmt.Errorf("option %q given, but not possible%s", opt.GetName(), desc)
			}
		}
	}
	return nil
}
