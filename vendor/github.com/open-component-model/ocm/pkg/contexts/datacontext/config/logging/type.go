// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"encoding/json"

	"github.com/mandelsoft/logging"
	logcfg "github.com/mandelsoft/logging/config"

	"github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/errors"
	local "github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ConfigType   = "logging" + cpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterConfigType(ConfigType, cpi.NewConfigType(ConfigType, &Config{}, usage))
	cpi.RegisterConfigType(ConfigTypeV1, cpi.NewConfigType(ConfigTypeV1, &Config{}, usage))
}

// Config describes logging settings for a dedicated context type.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`

	// ContextType described the context type to apply the setting.
	// If not set, the settings will be applied to any logging context provider,
	// which are not derived contexts.
	ContextType string        `json:"contextType,omitempty"`
	Settings    logcfg.Config `json:"settings"`

	// ExtraId is used to the context type "default" to be able
	// to reapply the same config again using a different
	// identity given by the settings hash + the id.
	ExtraId string `json:"extraId,omitempty"`
}

// NewConfigSpec creates a new memory ConfigSpec.
func New(ctxtype string, deflvl int) *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedObjectType(ConfigType),
		ContextType:         ctxtype,
		Settings: logcfg.Config{
			DefaultLevel: logging.LevelName(deflvl),
			Rules:        []json.RawMessage{},
		},
	}
}

func (c *Config) AddRuleSpec(spec interface{}) error {
	var err error

	data, ok := spec.([]byte)
	if !ok {
		data, err = json.Marshal(spec)
		if err != nil {
			errors.Wrapf(err, "invalid logging rule specification")
		}
	}
	_, err = logcfg.DefaultRegistry().CreateRule(data)
	if err != nil {
		return errors.Wrapf(err, "invalid logging rule specification")
	}
	c.Settings.Rules = append(c.Settings.Rules, data)
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

	// configure loogging context providers.
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
