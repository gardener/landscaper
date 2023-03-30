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
	cpi.RegisterConfigType(ConfigType, cpi.NewConfigType(ConfigType, &Config{}, usage))
	cpi.RegisterConfigType(ConfigTypeV1, cpi.NewConfigType(ConfigTypeV1, &Config{}, usage))
}

// Config describes a memory based repository interface.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	Configurations              []*cpi.GenericConfig `json:"configurations"`
}

// NewConfigSpec creates a new memory ConfigSpec.
func New() *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedObjectType(ConfigType),
		Configurations:      []*cpi.GenericConfig{},
	}
}

func (c *Config) AddConfig(cfg cpi.Config) error {
	g, err := cpi.ToGenericConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to convert cpi config to generic: %w", err)
	}

	c.Configurations = append(c.Configurations, g)

	return nil
}

func (c *Config) GetType() string {
	return ConfigType
}

func (c *Config) ApplyTo(ctx cpi.Context, target interface{}) error {
	if cctx, ok := target.(cpi.Context); ok {
		list := errors.ErrListf("applying generic config list")
		for i, cfg := range c.Configurations {
			sub := fmt.Sprintf("config entry %d", i)
			list.Add(cctx.ApplyConfig(cfg, sub))
		}
		return list.Result()
	}
	return nil
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to define a list
of arbitrary configuration specifications:

<pre>
    type: ` + ConfigType + `
    configurations:
      - type: &lt;any config type>
        ...
      ...
</pre>
`
