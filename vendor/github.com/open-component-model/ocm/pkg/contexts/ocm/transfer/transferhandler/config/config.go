// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package scriptoption

import (
	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer/transferhandler/standard"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ConfigType   = "transport.ocm" + cfgcpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigType, usage))
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigTypeV1, usage))
}

// Config describes a set of transport options.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	Recursive                   *bool    `json:"recursive,omitempty"`
	ResourcesByValue            *bool    `json:"resourcesByValue,omitempty"`
	LocalByValue                *bool    `json:"localResourcesByValue,omitempty"`
	SourcesByValue              *bool    `json:"sourcesByValue,omitempty"`
	KeepGlobalAccess            *bool    `json:"keepGlobalAccess,omitempty"`
	StopOnExisting              *bool    `json:"stopOnExistingVersion,omitempty"`
	Overwrite                   *bool    `json:"overwrite,omitempty"`
	OmitAccessTypes             []string `json:"omitAccessTypes,omitempty"`
}

// NewConfig creates a new memory ConfigSpec.
func NewConfig() *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedTypedObject(ConfigType),
	}
}

func (c *Config) GetType() string {
	return ConfigType
}

func (c *Config) ApplyTo(ctx cfgcpi.Context, target interface{}) error {
	if c.Recursive != nil {
		if opts, ok := target.(standard.RecursiveOption); ok {
			opts.SetRecursive(*c.Recursive)
		}
	}
	if c.ResourcesByValue != nil {
		if opts, ok := target.(standard.ResourcesByValueOption); ok {
			opts.SetResourcesByValue(*c.ResourcesByValue)
		}
	}
	if c.LocalByValue != nil {
		if opts, ok := target.(standard.LocalResourcesByValueOption); ok {
			opts.SetLocalResourcesByValue(*c.LocalByValue)
		}
	}
	if c.SourcesByValue != nil {
		if opts, ok := target.(standard.SourcesByValueOption); ok {
			opts.SetSourcesByValue(*c.SourcesByValue)
		}
	}
	if c.KeepGlobalAccess != nil {
		if opts, ok := target.(standard.KeepGlobalAccessOption); ok {
			opts.SetKeepGlobalAccess(*c.KeepGlobalAccess)
		}
	}
	if c.StopOnExisting != nil {
		if opts, ok := target.(standard.StopOnExistingVersionOption); ok {
			opts.SetStopOnExistingVersion(*c.StopOnExisting)
		}
	}
	if c.Overwrite != nil {
		if opts, ok := target.(standard.OverwriteOption); ok {
			opts.SetOverwrite(*c.Overwrite)
		}
	}
	if c.OmitAccessTypes != nil {
		if opts, ok := target.(standard.OmitAccessTypesOption); ok {
			opts.SetOmittedAccessTypes(c.OmitAccessTypes...)
		}
	}
	return nil
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to define transfer scripts:

<pre>
    type: ` + ConfigType + `
    recursive: true
    overwrite: true
    localResourcesByValue: false
    resourcesByValue: true
    sourcesByValue: false
    keepGlobalAccess: false
    stopOnExistingVersion: false
    omitAccessTypes:
    - s3
</pre>
`
