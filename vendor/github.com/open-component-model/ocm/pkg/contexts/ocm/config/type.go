// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/open-component-model/ocm/pkg/contexts/config"
	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ConfigType   = "ocm" + cfgcpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigType, usage))
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigTypeV1))
}

// Config describes a memory based config interface.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	Aliases                     map[string]*cpi.GenericRepositorySpec `json:"aliases,omitempty"`
	Resolver                    []ResolverRule                        `json:"resolvers,omitempty"`
}

type ResolverRule struct {
	Prefix string                     `json:"prefix,omitempty"`
	Prio   *int                       `json:"priority,omitempty"`
	Spec   *cpi.GenericRepositorySpec `json:"repository"`
}

// New creates a new memory ConfigSpec.
func New() *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedTypedObject(ConfigType),
	}
}

func (a *Config) GetType() string {
	return ConfigType
}

func (a *Config) SetAlias(name string, spec cpi.RepositorySpec) error {
	g, err := cpi.ToGenericRepositorySpec(spec)
	if err != nil {
		return err
	}
	if a.Aliases == nil {
		a.Aliases = map[string]*cpi.GenericRepositorySpec{}
	}
	a.Aliases[name] = g
	return nil
}

func (a *Config) AddResolverRule(prefix string, spec cpi.RepositorySpec, prio ...int) error {
	gen, err := cpi.ToGenericRepositorySpec(spec)
	if err != nil {
		return err
	}

	r := ResolverRule{
		Prefix: prefix,
		Spec:   gen,
	}
	if len(prio) > 0 {
		p := prio[0]
		r.Prio = &p
	}

	a.Resolver = append(a.Resolver, r)
	return nil
}

func (a *Config) ApplyTo(ctx config.Context, target interface{}) error {
	t, ok := target.(cpi.Context)
	if !ok {
		return config.ErrNoContext(ConfigType)
	}
	for n, s := range a.Aliases {
		t.SetAlias(n, s)
	}

	if len(a.Resolver) > 0 {
		for _, rule := range a.Resolver {
			if rule.Prio != nil {
				t.AddResolverRule(rule.Prefix, rule.Spec, *rule.Prio)
			} else {
				t.AddResolverRule(rule.Prefix, rule.Spec)
			}
		}
	}
	return nil
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to set some
configurations for an OCM context;

<pre>
    type: ` + ConfigType + `
    aliases:
       myrepo: 
          type: &lt;any repository type>
          &lt;specification attributes>
          ...
    resolvers:
      - repository:
          type: &lt;any repository type>
          &lt;specification attributes>
          ...
        prefix: ghcr.io/open-component-model/ocm
        priority: 10
</pre>

With aliases repository alias names can be mapped to a repository specification.
The alias name can be used in a string notation for an OCM repository.

Resolvers define a list of OCM repository specifications to be used to resolve
dedicated component versions. These settings are used to compose a standard
component version resolver provided for an OCM context. Optionally, a component
name prefix can be given. It limits the usage of the repository to resolve only
components with the given name prefix (always complete name segments).
An optional priority can be used to influence the lookup order. Larger value
means higher priority (default 10).

All matching entries are tried to lookup a component version in the following
order:
- highest priority first
- longest matching sequence of component name segments first.

If resolvers are defined, it is possible to use component version names on the
command line without a repository. The names are resolved with the specified
resolution rule.
They are also used as default lookup repositories to lookup component references
for recursive operations on component versions (<code>--lookup</code> option).
`
