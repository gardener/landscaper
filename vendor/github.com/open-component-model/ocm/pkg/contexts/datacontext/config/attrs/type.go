// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package attrs

import (
	"encoding/json"

	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ConfigType   = "attributes" + cfgcpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cfgcpi.RegisterConfigType(ConfigType, cfgcpi.NewConfigType(ConfigType, &Config{}, usage))
	cfgcpi.RegisterConfigType(ConfigTypeV1, cfgcpi.NewConfigType(ConfigTypeV1, &Config{}, usage))
}

// Config describes a memory based repository interface.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	// Attributes descibe a set of geeric attribute settings
	Attributes map[string]json.RawMessage `json:"attributes,omitempty"`
}

// New creates a new memory ConfigSpec.
func New() *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedObjectType(ConfigType),
		Attributes:          map[string]json.RawMessage{},
	}
}

func (a *Config) GetType() string {
	return ConfigType
}

func (a *Config) AddAttribute(attr string, value interface{}) error {
	data, err := datacontext.DefaultAttributeScheme.Encode(attr, value, runtime.DefaultJSONEncoding)
	if err == nil {
		a.Attributes[attr] = data
	}
	return err
}

func (a *Config) AddRawAttribute(attr string, data []byte) error {
	_, err := datacontext.DefaultAttributeScheme.Decode(attr, data, runtime.DefaultJSONEncoding)
	if err == nil {
		a.Attributes[attr] = data
	}
	return err
}

func (a *Config) ApplyTo(ctx cfgcpi.Context, target interface{}) error {
	list := errors.ErrListf("applying config")
	t, ok := target.(cfgcpi.Context)
	if !ok {
		return cfgcpi.ErrNoContext(ConfigType)
	}
	if a.Attributes == nil {
		return nil
	}
	for a, e := range a.Attributes {
		eff := datacontext.DefaultAttributeScheme.Shortcuts()[a]
		if eff != "" {
			a = eff
		}
		list.Add(errors.Wrapf(t.GetAttributes().SetEncodedAttribute(a, e, runtime.DefaultJSONEncoding), "attribute %q", a))
	}
	return list.Result()
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to define a list
of arbitrary attribute specifications:

<pre>
    type: ` + ConfigType + `
    attributes:
       &lt;name>: &lt;yaml defining the attribute>
       ...
</pre>
`
