// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ConfigType   = "generic" + cpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterConfigType(cpi.NewConfigType[*Config](ConfigType, usage))
	cpi.RegisterConfigType(cpi.NewConfigType[*Config](ConfigTypeV1, usage))
}

// Config describes a memory based repository interface.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	cpi.ConfigurationList       `json:",inline"`
	Sets                        map[string]cpi.ConfigSet `json:"sets,omitempty"`
}

// New creates a new memory ConfigSpec.
func New() *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedTypedObject(ConfigType),
		ConfigurationList:   cpi.ConfigurationList{[]*cpi.GenericConfig{}},
		Sets:                map[string]cpi.ConfigSet{},
	}
}

func (c *Config) AddSet(name, desc string) {
	set := c.Sets[name]
	set.Description = desc
	c.Sets[name] = set
}

func (c *Config) AddConfigToSet(name string, cfg cpi.Config) error {
	set := c.Sets[name]
	err := set.AddConfig(cfg)
	if err == nil {
		c.Sets[name] = set
	}
	return err
}

func (c *Config) GetType() string {
	return ConfigType
}

func (c *Config) ApplyTo(ctx cpi.Context, target interface{}) error {
	if cctx, ok := target.(cpi.Context); ok {
		for n, s := range c.Sets {
			set := s
			cctx.AddConfigSet(n, &set)
		}

		list := errors.ErrListf("applying generic config list")
		for i, cfg := range c.Configurations {
			sub := fmt.Sprintf("config entry %d", i)
			list.Add(cctx.ApplyConfig(cfg, ctx.WithInfo(sub).Info()))
		}
		return list.Result()
	}
	return cpi.ErrNoContext(ConfigType)
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to define a list
of arbitrary configuration specifications and named configuration sets:

<pre>
    type: ` + ConfigType + `
    configurations:
      - type: &lt;any config type>
        ...
      ...
    sets:
       standard:
          description: my selectable standard config
          configurations:
            - type: ...
              ...
            ...
</pre>

Configurations are directly applied. Configuration sets are
just stored in the configuration context and can be applied
on-demand. On the CLI, this can be done using the main command option
<code>--config-set &lt;name></code>.
`
