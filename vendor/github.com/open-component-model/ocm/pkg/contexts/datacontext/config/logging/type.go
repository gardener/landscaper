// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"github.com/mandelsoft/logging"
	logcfg "github.com/mandelsoft/logging/config"

	"github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	local "github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ConfigType   = "logging" + cpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterConfigType(cpi.NewConfigType[*Config](ConfigType, usage))
	cpi.RegisterConfigType(cpi.NewConfigType[*Config](ConfigTypeV1, usage))
}

// Config describes logging settings for a dedicated context type.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`

	// ContextType described the context type to apply the setting.
	// If not set, the settings will be applied to any logging context provider,
	// which are not derived contexts.
	ContextType string        `json:"contextType,omitempty"`
	Settings    logcfg.Config `json:"settings"`

	// ExtraId is used to the context type "default" or "global" to be able
	// to reapply the same config again using a different
	// identity given by the settings hash + the id.
	ExtraId string `json:"extraId,omitempty"`
}

// New creates a logging config specification.
func New(ctxtype string, deflvl int) *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedTypedObject(ConfigType),
		ContextType:         ctxtype,
		Settings: logcfg.Config{
			DefaultLevel: logging.LevelName(deflvl),
		},
	}
}

// NewWithConfig creates a logging config specification from a
// logging config object.
func NewWithConfig(ctxtype string, cfg *logcfg.Config) *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedTypedObject(ConfigType),
		ContextType:         ctxtype,
		Settings:            *cfg,
	}
}

func (c *Config) AddRuleSpec(r logcfg.Rule) error {
	c.Settings.Rules = append(c.Settings.Rules, r)
	return nil
}

func (c *Config) GetType() string {
	return ConfigType
}

func (c *Config) ApplyTo(ctx cpi.Context, target interface{}) error {
	lctx, ok := target.(logging.ContextProvider)
	if !ok {
		return cpi.ErrNoContext("logging context")
	}

	switch c.ContextType {
	// configure local static logging context.
	// here, config is only applied once for every
	// setting hash.
	case "default":
		return local.Configure(&c.Settings, c.ExtraId)

	case "global":
		return local.ConfigureGlobal(&c.Settings, c.ExtraId)

	// configure logging context providers.
	case "":
		if _, ok := target.(datacontext.AttributesContext); !ok {
			return cpi.ErrNoContext("attribute context")
		}

	// configure dedicated context types.
	default:
		dc, ok := target.(datacontext.Context)
		if !ok {
			return cpi.ErrNoContext("data context")
		}
		if dc.GetType() != c.ContextType {
			return cpi.ErrNoContext(c.ContextType)
		}
	}
	return logcfg.DefaultRegistry().Configure(lctx.LoggingContext(), &c.Settings)
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to configure the logging
aspect of a dedicated context type:

<pre>
    type: ` + ConfigType + `
    contextType: ` + datacontext.CONTEXT_TYPE + `
    settings:
      defaultLevel: Info
      rules:
        - ...
</pre>

The context type ` + datacontext.CONTEXT_TYPE + ` is the root context of a
context hierarchy.

If no context type is specified, the config will be applies to any target
acting as logging context provider, which is not a non-root context.
`
