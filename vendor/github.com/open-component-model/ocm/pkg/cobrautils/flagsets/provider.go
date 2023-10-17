// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsets

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
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

type _ConfigTypeOptionSetConfigProvider = ConfigTypeOptionSetConfigProvider

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
	_ConfigTypeOptionSetConfigProvider
	typeOption string
}

var _ ConfigTypeOptionSetConfigProvider = (*typedConfigProvider)(nil)

func NewTypedConfigProvider(name string, desc, typeOption string, acceptUnknown ...bool) ConfigTypeOptionSetConfigProvider {
	return &typedConfigProvider{NewTypedConfigProviderBase(name, desc, TypeNameProviderFromOptions(typeOption), utils.Optional(acceptUnknown...), NewStringOptionType(name+"Type", "type of "+desc)), typeOption}
}

func (p *typedConfigProvider) IsExplicitlySelected(opts ConfigOptions) bool {
	return opts.Changed(p.typeOption, p.GetName())
}

func TypeNameProviderFromOptions(name string) TypeNameProvider {
	return func(opts ConfigOptions) (string, error) {
		typv, _ := opts.GetValue(name)
		typ, ok := typv.(string)
		if !ok {
			return "", fmt.Errorf("failed to assert type %T as string", typv)
		}
		return typ, nil
	}
}

///////////////////////////////////////////////////////////////////////////////

type ExplicitlyTypedConfigTypeOptionSetConfigProvider interface {
	ConfigTypeOptionSetConfigProvider
	SetTypeName(n string)
}

type explicitlyTypedConfigProvider struct {
	_ConfigTypeOptionSetConfigProvider
	typeName string
}

var _ ConfigTypeOptionSetConfigProvider = (*typedConfigProvider)(nil)

func NewExplicitlyTypedConfigProvider(name string, desc string, acceptUnknown ...bool) ExplicitlyTypedConfigTypeOptionSetConfigProvider {
	p := &explicitlyTypedConfigProvider{}
	p._ConfigTypeOptionSetConfigProvider = NewTypedConfigProviderBase(name, desc, p.getTypeName, utils.Optional(acceptUnknown...))
	return p
}

func (p *explicitlyTypedConfigProvider) SetTypeName(n string) {
	p.typeName = n
}

func (p *explicitlyTypedConfigProvider) getTypeName(opts ConfigOptions) (string, error) {
	return p.typeName, nil
}

////////////////////////////////////////////////////////////////////////////////

type TypeNameProvider func(opts ConfigOptions) (string, error)

type typedConfigProviderBase struct {
	ConfigOptionTypeSet
	typeProvider  TypeNameProvider
	meta          ConfigOptionTypeSet
	acceptUnknown bool
}

var _ ConfigTypeOptionSetConfigProvider = (*typedConfigProviderBase)(nil)

func NewTypedConfigProviderBase(name string, desc string, prov TypeNameProvider, acceptUnknown bool, types ...ConfigOptionType) ConfigTypeOptionSetConfigProvider {
	set := NewConfigOptionTypeSet(name, append(types, NewValueMapYAMLOptionType(name, desc+" (YAML)"))...)
	return &typedConfigProviderBase{
		ConfigOptionTypeSet: set,
		typeProvider:        prov,
		meta:                NewConfigOptionTypeSet(name, append(types, NewValueMapYAMLOptionType(name, desc+" (YAML)"))...),
		acceptUnknown:       acceptUnknown,
	}
}

func (p *typedConfigProviderBase) GetConfigOptionTypeSet() ConfigOptionTypeSet {
	return p
}

func (p *typedConfigProviderBase) IsExplicitlySelected(opts ConfigOptions) bool {
	t, err := p.typeProvider(opts)
	return err == nil && t != ""
}

func (p *typedConfigProviderBase) GetConfigFor(opts ConfigOptions) (Config, error) {
	typ, err := p.typeProvider(opts)
	if err != nil {
		return nil, err
	}
	cfgv, _ := opts.GetValue(p.GetName())

	var data Config
	if cfgv != nil {
		var ok bool
		data, ok = cfgv.(Config)
		if !ok {
			return nil, fmt.Errorf("failed to assert type %T as Config", cfgv)
		}
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
		data["type"] = typ
		if err := p.applyConfigForType(typ, opts, data); err != nil {
			if !p.acceptUnknown || !errors.Is(err, errors.ErrUnknown(typ)) {
				return nil, err
			}
			unexpected := opts.FilterBy(And(Changed(opts), Not(p.meta.HasOptionType))).Names()
			if len(unexpected) > 0 {
				return nil, fmt.Errorf("unexpected options %s", strings.Join(unexpected, ", "))
			}
		}
	}
	return data, nil
}

func (p *typedConfigProviderBase) GetTypeSetForType(name string) ConfigOptionTypeSet {
	set := p.GetTypeSet(name)
	if set == nil {
		k, v := runtime.KindVersion(name)
		if v == "" {
			set = p.GetTypeSet(runtime.TypeName(name, "v1"))
		} else if v == "v1" {
			set = p.GetTypeSet(k)
		}
	}
	return set
}

func (p *typedConfigProviderBase) applyConfigForType(name string, opts ConfigOptions, config Config) error {
	set := p.GetTypeSetForType(name)
	if set == nil {
		return errors.ErrUnknown(name)
	}

	opts = opts.FilterBy(Not(p.meta.HasOptionType))
	err := opts.Check(set, p.GetName()+" type "+name)
	if err != nil {
		return err
	}
	handler, ok := set.(ConfigHandler)
	if !ok {
		return fmt.Errorf("no config handler defined for %s type %s", p.GetName(), name)
	}
	return handler.ApplyConfig(opts, config)
}
