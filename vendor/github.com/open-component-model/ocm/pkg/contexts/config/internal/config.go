// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const KIND_CONFIGSET = "config set"

type Config interface {
	runtime.VersionedTypedObject

	ApplyTo(Context, interface{}) error
}

type ConfigSet struct {
	Description       string `json:"description,omitempty"`
	ConfigurationList `json:",inline"`
}

type ConfigurationList struct {
	Configurations []*GenericConfig `json:"configurations,omitempty"`
}

func (c *ConfigurationList) AddConfig(cfg Config) error {
	g, err := ToGenericConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to convert config to generic: %w", err)
	}

	c.Configurations = append(c.Configurations, g)

	return nil
}

func (c *ConfigurationList) AddConfigData(ctx Context, data []byte) error {
	cfg, err := ctx.GetConfigForData(data, nil)
	if err != nil {
		return errors.Wrapf(err, "invalid config specification")
	}
	g, err := ToGenericConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to convert config to generic: %w", err)
	}

	c.Configurations = append(c.Configurations, g)
	return nil
}
