// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsets

import (
	"strings"
)

// Config is a generic structured config stored in a string map.
type Config = map[string]interface{}

// ConfigAdder is used to incorporate a partial config into an existing one.
type ConfigAdder func(options ConfigOptions, config Config) error

func (c ConfigAdder) ApplyConfig(options ConfigOptions, config Config) error {
	return c(options, config)
}

// ConfigHandler describes the ConfigAdder functionality.
type ConfigHandler interface {
	ApplyConfig(options ConfigOptions, config Config) error
}

// ConfigOptionTypeSetHandler describes a ConfigOptionTypeSet, which also
// provides the possibility to provide config.
type ConfigOptionTypeSetHandler interface {
	ConfigOptionTypeSet
	ConfigHandler
}

type configOptionTypeSetHandler struct {
	adder ConfigAdder
	ConfigOptionTypeSet
}

// NewConfigOptionTypeSetHandler creates a new ConfigOptionTypeSetHandler.
func NewConfigOptionTypeSetHandler(name string, adder ConfigAdder, types ...ConfigOptionType) ConfigOptionTypeSetHandler {
	return &configOptionTypeSetHandler{
		adder:               adder,
		ConfigOptionTypeSet: NewConfigOptionTypeSet(name, types...),
	}
}

func (c *configOptionTypeSetHandler) ApplyConfig(options ConfigOptions, config Config) error {
	if c.adder == nil {
		return nil
	}
	return c.adder(options, config)
}

type nopConfigHandler struct{}

// NopConfigHandler is a dummy config handler doing nothing.
var NopConfigHandler = NewNopConfigHandler()

func NewNopConfigHandler() ConfigHandler {
	return &nopConfigHandler{}
}

func (c *nopConfigHandler) ApplyConfig(options ConfigOptions, config Config) error {
	return nil
}

func FormatConfigOptions(handler ConfigOptionTypeSetHandler) string {
	group := ""
	if handler != nil {
		opts := handler.OptionTypeNames()
		var names []string
		if len(opts) > 0 {
			for _, o := range opts {
				names = append(names, "<code>--"+o+"</code>")
			}
			group = "\nOptions used to configure fields: " + strings.Join(names, ", ") + "\n"
		}
	}
	return group
}
