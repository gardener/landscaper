// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package hashattr

import (
	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/listformat"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/hasher/sha256"
)

const (
	ConfigType   = "hasher" + cfgcpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigType, usage))
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigTypeV1, usage))
}

// Config describes a memory based repository interface.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	HashAlgorithm               string `json:"hashAlgorithm"`
}

// New creates a new memory ConfigSpec.
func New(algo string) *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedTypedObject(ConfigType),
		HashAlgorithm:       algo,
	}
}

func (a *Config) GetType() string {
	return ConfigType
}

func (a *Config) ApplyTo(ctx cfgcpi.Context, target interface{}) error {
	t, ok := target.(Context)
	if !ok {
		return cfgcpi.ErrNoContext(ConfigType)
	}
	return errors.Wrapf(t.GetAttributes().SetAttribute(ATTR_KEY, a.HashAlgorithm), "applying config failed")
}

var usage = `
The config type <code>` + ConfigType + `</code> can be used to define
the default hash algorithm used to calculate digests for resources.
It supports the field <code>hashAlgorithm</code>, with one of the following
values:
` + listformat.FormatList(sha256.Algorithm, signing.DefaultRegistry().HasherNames()...)
