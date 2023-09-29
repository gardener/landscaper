// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package defaultmerge

import (
	// special case to resolve dependency cycles.
	hpi "github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/internal"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Mode string

func (m Mode) String() string {
	return string(m)
}

const (
	MODE_DEFAULT = Mode("")
	MODE_NONE    = Mode("none")
	MODE_LOCAL   = Mode("local")
	MODE_INBOUND = Mode("inbound")
)

func NewConfig(overwrite Mode) *Config {
	return &Config{
		Overwrite: overwrite,
	}
}

type Config struct {
	Overwrite Mode `json:"overwrite"`
}

func (c Config) Complete(ctx hpi.Context) error {
	switch c.Overwrite {
	case MODE_NONE, MODE_LOCAL, MODE_INBOUND:
	case "":
		// leave choice to using algorithm
	default:
		return errors.ErrInvalid("merge overwrite mode", string(c.Overwrite))
	}
	return nil
}
