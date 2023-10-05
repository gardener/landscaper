// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package maplistmerge

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/handlers/defaultmerge"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/hpi"
	"github.com/open-component-model/ocm/pkg/utils"
)

type Mode = defaultmerge.Mode

const (
	MODE_DEFAULT = defaultmerge.MODE_DEFAULT
	MODE_NONE    = defaultmerge.MODE_NONE
	MODE_LOCAL   = defaultmerge.MODE_LOCAL
	MODE_INBOUND = defaultmerge.MODE_INBOUND
)

func NewConfig(field string, overwrite Mode, entries ...*hpi.Specification) *Config {
	return &Config{
		KeyField: field,
		Config:   *defaultmerge.NewConfig(overwrite),
		Entries:  utils.Optional(entries...),
	}
}

type Config struct {
	defaultmerge.Config
	KeyField string             `json:"keyField"`
	Entries  *hpi.Specification `json:"entries,omitempty"`
}

func (c *Config) Complete(ctx hpi.Context) error {
	err := c.Config.Complete(ctx)
	if err != nil {
		return err
	}
	if c.KeyField == "" {
		c.KeyField = "name"
	}
	return nil
}
