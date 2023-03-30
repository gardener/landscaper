// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsets

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/errors"
)

type ConfigProvider interface {
	CreateOptions() ConfigOptions
	GetConfigFor(opts ConfigOptions) (Config, error)
}

type ConfigTypeOptionSetConfigProvider interface {
	ConfigProvider
	ConfigOptionTypeSet

	IsExplicitlySelected(opts ConfigOptions) bool
}

////////////////////////////////////////////////////////////////////////////////

type plainConfigProvider struct {
	ConfigOptionTypeSetHandler
}

var _ ConfigTypeOptionSetConfigProvider = (*plainConfigProvider)(nil)

func NewPlainConfigProvider(name string, adder ConfigAdder, types ...ConfigOptionType) ConfigTypeOptionSetConfigProvider {
	h := NewConfigOptionTypeSetHandler(name, adder, types...)
	return &plainConfigProvider{
		ConfigOptionTypeSetHandler: h,
	}
}

func (p *plainConfigProvider) GetConfigOptionTypeSet() ConfigOptionTypeSet {
	return p
}

func (p *plainConfigProvider) IsExplicitlySelected(opts ConfigOptions) bool {
	return opts.FilterBy(p.HasOptionType).Changed()
}

func (p *plainConfigProvider) GetConfigFor(opts ConfigOptions) (Config, error) {
	if !p.IsExplicitlySelected(opts) {
		return nil, nil
	}
	config := Config{}
	err := p.ApplyConfig(opts, config)
	return config, err
}

////////////////////////////////////////////////////////////////////////////////

type typedConfigProvider struct {
	ConfigOptionTypeSet
}

var _ ConfigTypeOptionSetConfigProvider = (*typedConfigProvider)(nil)

func NewTypedConfigProvider(name string, desc string) ConfigTypeOptionSetConfigProvider {
	set := NewConfigOptionSet(name, NewValueMapYAMLOptionType(name, desc+" (YAML)"), NewStringOptionType(name+"Type", "type of "+desc))
	return &typedConfigProvider{
		ConfigOptionTypeSet: set,
	}
}

func (p *typedConfigProvider) GetConfigOptionTypeSet() ConfigOptionTypeSet {
	return p
}

func (p *typedConfigProvider) IsExplicitlySelected(opts ConfigOptions) bool {
	return opts.Changed(p.GetName()+"Type", p.GetName())
}

func (p *typedConfigProvider) GetConfigFor(opts ConfigOptions) (Config, error) {
	typv, _ := opts.GetValue(p.GetName() + "Type")
	cfgv, _ := opts.GetValue(p.GetName())

	var data Config
	if cfgv != nil {
		var ok bool
		data, ok = cfgv.(Config)
		if !ok {
			return nil, fmt.Errorf("failed to assert type %T as Config", cfgv)
		}
	}
	typ, ok := typv.(string)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T as string", typv)
	}

	opts = opts.FilterBy(p.HasOptionType)
	if typ == "" && data != nil && data["type"] != nil {
		t := data["type"]
		if t != nil {
			if s, ok := t.(string); ok {
				typ = s
			} else {
				return nil, fmt.Errorf("type field must be a string")
			}
		}
	}

	if opts.Changed() || typ != "" {
		if typ == "" {
			return nil, fmt.Errorf("type required for explicitly configured options")
		}
		if data == nil {
			data = Config{}
		}
		if typ != "" {
			data["type"] = typ
		}
		if err := p.applyConfigForType(typ, opts, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (p *typedConfigProvider) applyConfigForType(name string, opts ConfigOptions, config Config) error {
	set := p.GetTypeSet(name)
	if set == nil {
		return errors.ErrUnknown(name)
	}

	err := opts.FilterBy(p.HasSharedOptionType).Check(set, p.GetName()+" type "+name)
	if err != nil {
		return err
	}
	handler, ok := set.(ConfigHandler)
	if !ok {
		return fmt.Errorf("no config handler defined for %s type %s", p.GetName(), name)
	}
	return handler.ApplyConfig(opts, config)
}
